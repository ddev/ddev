package ddevapp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
)

// RouterComposeProjectName is the docker-compose project name of ~/.ddev/.router-compose.yaml
const (
	RouterComposeProjectName = "ddev-router"
	MinEphemeralPort         = 33000
	MaxEphemeralPort         = 35000
)

// EphemeralRouterPortsAssigned is used when we have assigned an ephemeral port
// but it may not yet be occupied. A map is used just to make it easy
// to detect if it's there, the value in the map is not used.
var EphemeralRouterPortsAssigned = make(map[int]bool)

// RouterComposeYAMLPath returns the full filepath to the routers docker-compose yaml file.
func RouterComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".router-compose.yaml")
	return dest
}

// FullRenderedRouterComposeYAMLPath returns the path of the full rendered .router-compose-full.yaml
func FullRenderedRouterComposeYAMLPath() string {
	globalDir := globalconfig.GetGlobalDdevDir()
	dest := path.Join(globalDir, ".router-compose-full.yaml")
	return dest
}

// IsRouterDisabled returns true if the router is disabled
func IsRouterDisabled(app *DdevApp) bool {
	if nodeps.IsCodespaces() {
		return true
	}
	return nodeps.ArrayContainsString(app.GetOmittedContainers(), globalconfig.DdevRouterContainer)
}

// StopRouterIfNoContainers stops the router if there are no DDEV containers running.
func StopRouterIfNoContainers() error {
	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		routerPorts, err := GetRouterBoundPorts()
		if err != nil {
			return err
		}
		util.Debug("stopping ddev-router because all project containers are stopped")
		err = dockerutil.RemoveContainer(nodeps.RouterContainer)
		if err != nil {
			if ok := dockerutil.IsErrNotFound(err); !ok {
				return err
			}
		}

		// Colima and Lima don't release ports very fast after container is removed
		// see https://github.com/lima-vm/lima/issues/2536 and
		// https://github.com/abiosoft/colima/issues/644
		if dockerutil.IsLima() || dockerutil.IsColima() || dockerutil.IsRancherDesktop() {
			if globalconfig.DdevDebug {
				util.Debug("Lima/Colima/Rancher stopping router")
				dockerContainers, _ := dockerutil.GetDockerContainers(true)
				containerInfo := make([]string, len(dockerContainers))
				for i, c := range dockerContainers {
					containerInfo[i] = fmt.Sprintf("ID: %s, Name: %s, State: %s, Image: %s", dockerutil.TruncateID(c.ID), dockerutil.ContainerName(&c), c.State, c.Image)
				}
				containerList, _ := util.ArrayToReadableOutput(containerInfo)
				util.Debug("All docker containers: %s", containerList)
			}
			util.Debug("Waiting for router ports to be released on Lima-based systems because ports aren't released immediately")
			waitForPortsToBeReleased(routerPorts, time.Second*5)
			// Wait another couple of seconds
			time.Sleep(time.Second * 2)
		}
	}
	return nil
}

// waitForPortsToBeReleased waits until the specified ports are released or the timeout is reached.
func waitForPortsToBeReleased(ports []uint16, timeout time.Duration) {
	util.Debug("starting port release for ports: %v", ports)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500 milliseconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			util.Debug("Timeout reached, stopping check.")
			return
		case <-ticker.C:
			allReleased := true
			for _, portInt := range ports {
				port := fmt.Sprintf("%d", portInt)
				if netutil.IsPortActive(port) {
					util.Debug("Port %s is still in use.", port)
					allReleased = false
				} else {
					util.Debug("Port %s is released.", port)
				}
			}
			if allReleased {
				util.Debug("All ports are released.")
				return
			}
		}
	}
}

// StartDdevRouter ensures the router is running.
func StartDdevRouter() error {
	// If the router is not healthy/running, we'll kill it so it
	// starts over again.
	router, err := FindDdevRouter()
	if router != nil && err == nil && router.State != "running" {
		err = dockerutil.RemoveContainer(nodeps.RouterContainer)
		if err != nil {
			return err
		}
	}

	activeApps := GetActiveProjects()
	routerComposeFullPath, err := generateRouterCompose(activeApps)
	if err != nil {
		return err
	}
	err = PushGlobalTraefikConfig(activeApps)
	if err != nil {
		return fmt.Errorf("failed to push global Traefik config: %v", err)
	}

	err = CheckRouterPorts(activeApps)
	if err != nil {
		return fmt.Errorf("unable to listen on required ports, %v,\nTroubleshooting suggestions at https://docs.ddev.com/en/stable/users/usage/troubleshooting/#unable-listen", err)
	}

	// Run docker-compose up -d against the ddev-router full compose file
	_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: []string{routerComposeFullPath},
		Action:       []string{"-p", RouterComposeProjectName, "up", "--build", "-d"},
	})
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	// Ensure we have a happy router
	label := map[string]string{
		"com.docker.compose.service": nodeps.RouterContainer,
		"com.docker.compose.oneoff":  "False",
	}
	// Normally the router comes right up, but when
	// it has to do let's encrypt updates, it can take
	// some time.
	routerWaitTimeout := 60
	if globalconfig.DdevGlobalConfig.UseLetsEncrypt {
		routerWaitTimeout = 180
	}
	// Print without newline so we can append elapsed time on the same line
	if !output.JSONOutput {
		_, _ = fmt.Fprintf(os.Stdout, "Waiting for %s to become ready...", nodeps.RouterContainer)
	}
	util.Debug("Router wait: checking for container with labels %v, polling every 500ms for healthy status", label)
	startTime := time.Now()
	logOutput, err := dockerutil.ContainerWait(routerWaitTimeout, label)
	elapsed := time.Since(startTime)
	if err != nil {
		if !output.JSONOutput {
			_, _ = fmt.Fprintf(os.Stdout, "\n")
		}
		return fmt.Errorf("ddev-router failed to become ready after %.1fs; log=%s, err=%v", elapsed.Seconds(), logOutput, err)
	}
	if !output.JSONOutput {
		_, _ = fmt.Fprintf(os.Stdout, " ready in %.1fs, %s\n", elapsed.Seconds(), logOutput)
	}

	util.Debug("Getting traefik error output")
	traefikErr := GetRouterConfigErrors()
	if traefikErr != "" {
		util.Warning("Warning: There are router configuration problems:\n%s", traefikErr)
		util.Warning("For help go to: https://docs.ddev.com/en/stable/users/extend/traefik-router/#troubleshooting-traefik-routing")
	}

	return nil
}

// generateRouterCompose() generates the ~/.ddev/.router-compose.yaml and ~/.ddev/.router-compose-full.yaml
func generateRouterCompose(activeApps []*DdevApp) (string, error) {
	exposedPorts := determineRouterPorts(activeApps)

	routerComposeBasePath := RouterComposeYAMLPath()
	routerComposeFullPath := FullRenderedRouterComposeYAMLPath()

	var doc bytes.Buffer
	f, ferr := os.Create(routerComposeBasePath)
	if ferr != nil {
		return "", ferr
	}
	defer util.CheckClose(f)

	dockerIP, _ := dockerutil.GetDockerIP()

	uid, gid, username := dockerutil.GetContainerUser()
	timezone, _ := util.GetLocalTimezone()

	templateVars := map[string]interface{}{
		"Username":                   username,
		"UID":                        uid,
		"GID":                        gid,
		"router_image":               ddevImages.GetRouterImage(),
		"ports":                      exposedPorts,
		"router_bind_all_interfaces": globalconfig.DdevGlobalConfig.RouterBindAllInterfaces,
		"dockerIP":                   dockerIP,
		"letsencrypt":                globalconfig.DdevGlobalConfig.UseLetsEncrypt,
		"letsencrypt_email":          globalconfig.DdevGlobalConfig.LetsEncryptEmail,
		"Router":                     globalconfig.DdevGlobalConfig.Router,
		"TraefikMonitorPort":         globalconfig.DdevGlobalConfig.TraefikMonitorPort,
		"Timezone":                   timezone,
		"Hostnames":                  determineRouterHostnames(activeApps),
		"IsPodman":                   dockerutil.IsPodman(),
		"IsRootless":                 dockerutil.IsRootless(),
	}

	t, err := template.New("router_compose_template.yaml").ParseFS(bundledAssets, "router_compose_template.yaml")
	if err != nil {
		return "", err
	}

	err = t.Execute(&doc, templateVars)
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(doc.String())
	if err != nil {
		return "", err
	}

	fullHandle, err := os.Create(routerComposeFullPath)
	if err != nil {
		return "", err
	}

	userFiles, err := filepath.Glob(filepath.Join(globalconfig.GetGlobalDdevDir(), "router-compose.*.yaml"))
	if err != nil {
		return "", err
	}
	files := append([]string{RouterComposeYAMLPath()}, userFiles...)
	fullContents, _, err := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
		ComposeFiles: files,
		Action:       []string{"config"},
	})
	if err != nil {
		return "", err
	}
	_, err = fullHandle.WriteString(fullContents)
	if err != nil {
		return "", err
	}

	return routerComposeFullPath, nil
}

// FindDdevRouter uses FindContainerByLabels to get our router container and
// return it.
func FindDdevRouter() (*container.Summary, error) {
	containerQuery := map[string]string{
		"com.docker.compose.service": nodeps.RouterContainer,
		"com.docker.compose.oneoff":  "False",
	}
	c, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	if c == nil {
		return nil, fmt.Errorf("no ddev-router was found")
	}
	return c, nil
}

// GetRouterBoundPorts returns the currently bound ports on ddev-router
// or an empty array if router not running
func GetRouterBoundPorts() ([]uint16, error) {
	boundPorts := []uint16{}
	r, err := FindDdevRouter()
	if err != nil {
		return []uint16{}, nil
	}

	for _, p := range r.Ports {
		if p.PublicPort != 0 {
			boundPorts = append(boundPorts, p.PublicPort)
		}
	}
	return boundPorts, nil
}

// RenderRouterStatus returns a user-friendly string showing router-status
func RenderRouterStatus() (string, string) {
	var renderedStatus, errorInfo string
	if !nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, globalconfig.DdevRouterContainer) {
		status, logOutput := GetRouterStatus()

		switch status {
		case SiteStopped:
			renderedStatus = util.ColorizeText(status, "red")
		case "exited", string(container.Unhealthy):
			renderedStatus = util.ColorizeText(status, "red")
			errorInfo = "The router is not healthy, your projects may not be accessible, " +
				"if it doesn't become healthy, run 'ddev poweroff && ddev start' on a project to recreate it."
			if logOutput != "" {
				errorInfo = errorInfo + "\n" + logOutput
			}
		case string(container.Healthy):
			status = "OK"
			// If there are router configuration errors, show them
			if configErrors := GetRouterConfigErrors(); configErrors != "" {
				lines := strings.Split(configErrors, "\n")
				errorInfo = fmt.Sprintf("Detected %d configuration error(s):\n%s", len(lines), configErrors)
			}
			fallthrough
		default:
			renderedStatus = util.ColorizeText(status, "green")
		}
	} else {
		renderedStatus = "disabled"
	}
	return renderedStatus, errorInfo
}

// GetRouterStatus returns router status and warning if not
// running or healthy, as applicable.
// return status and most recent log
func GetRouterStatus() (string, string) {
	var status, logOutput string
	c, err := FindDdevRouter()

	if err != nil || c == nil {
		status = SiteStopped
	} else {
		status, logOutput = dockerutil.GetContainerHealth(c)
	}

	return status, logOutput
}

// GetRouterConfigErrors reads traefik configuration errors from the router container
func GetRouterConfigErrors() string {
	router, err := FindDdevRouter()
	if err != nil || router == nil {
		return ""
	}

	traefikErr, _, _ := dockerutil.Exec(router.ID, "cat /tmp/ddev-traefik-errors.txt 2>/dev/null || true", "0")
	return strings.TrimSpace(traefikErr)
}

// determineRouterHostnames returns a list of all hostnames for all active projects
func determineRouterHostnames(activeApps []*DdevApp) []string {
	var routerHostnames []string

	for _, app := range activeApps {
		_, hostnames, err := detectAppRouting(app)
		if err != nil {
			util.Verbose("Unable to determine hostnames for '%s' project: %v", app.Name, err)
			continue
		}
		routerHostnames = append(routerHostnames, hostnames...)
	}
	routerHostnames = util.SliceToUniqueSlice(&routerHostnames)
	return routerHostnames
}

// determineRouterPorts returns a list of port mappings retrieved from ports from
// configuration files of all active projects, plus running site
// containers defining HTTP_EXPOSE and HTTPS_EXPOSE env var.
// It is only useful to call this when containers are actually running, before
// starting ddev-router (so that we can bind the port mappings needed
func determineRouterPorts(activeApps []*DdevApp) []string {
	var routerPorts []string

	// Add ports from configuration files of all active projects
	routerPorts = append(routerPorts, getConfigBasedRouterPorts(activeApps)...)

	// Add ports from running containers
	routerPorts = append(routerPorts, getContainerBasedRouterPorts()...)

	// Remove duplicates
	seen := make(map[string]bool)
	var uniquePorts []string
	for _, port := range routerPorts {
		if !seen[port] {
			seen[port] = true
			uniquePorts = append(uniquePorts, port)
		}
	}

	sort.Slice(uniquePorts, func(i, j int) bool {
		return uniquePorts[i] < uniquePorts[j]
	})

	return uniquePorts
}

// getConfigBasedRouterPorts collects port mappings from configuration files of all active projects
func getConfigBasedRouterPorts(activeApps []*DdevApp) []string {
	var routerPorts []string

	for _, app := range activeApps {
		if app.ComposeYaml == nil || app.ComposeYaml.Services == nil {
			continue
		}
		// Extract ports from compose services
		for _, service := range app.ComposeYaml.Services {
			if service.Environment == nil {
				continue
			}

			var exposePorts []string

			if httpExposePtr, ok := service.Environment["HTTP_EXPOSE"]; ok && httpExposePtr != nil {
				if httpExpose := *httpExposePtr; httpExpose != "" {
					ports := strings.Split(httpExpose, ",")
					exposePorts = append(exposePorts, ports...)
				}
			}

			if httpsExposePtr, ok := service.Environment["HTTPS_EXPOSE"]; ok && httpsExposePtr != nil {
				if httpsExpose := *httpsExposePtr; httpsExpose != "" {
					ports := strings.Split(httpsExpose, ",")
					exposePorts = append(exposePorts, ports...)
				}
			}

			routerPorts = ProcessExposePorts(exposePorts, routerPorts)
		}
	}

	return routerPorts
}

// getContainerBasedRouterPorts collects port mappings from running site containers
func getContainerBasedRouterPorts() []string {
	var routerPorts []string

	containers, err := dockerutil.FindContainersWithLabel("com.ddev.site-name")
	if err != nil {
		util.Failed("Failed to retrieve containers for determining port mappings: %v", err)
	}

	for _, c := range containers {
		if _, ok := c.Labels["com.ddev.site-name"]; ok {
			if c.State != "running" {
				continue
			}
			var exposePorts []string

			httpPorts := dockerutil.GetContainerEnv("HTTP_EXPOSE", c)
			if httpPorts != "" {
				ports := strings.Split(httpPorts, ",")
				exposePorts = append(exposePorts, ports...)
			}

			httpsPorts := dockerutil.GetContainerEnv("HTTPS_EXPOSE", c)
			if httpsPorts != "" {
				ports := strings.Split(httpsPorts, ",")
				exposePorts = append(exposePorts, ports...)
			}

			routerPorts = ProcessExposePorts(exposePorts, routerPorts)
		}
	}

	return routerPorts
}

// ProcessExposePorts processes HTTP_EXPOSE and HTTPS_EXPOSE port strings and returns
// a list of external ports that need to be bound by the router.
// It handles port pair formats like "8080:80" or "8080" and validates the format.
func ProcessExposePorts(exposePorts []string, routerPorts []string) []string {
	for _, exposePortPair := range exposePorts {
		// Ports defined as hostPort:containerPort allow for router to configure upstreams
		// for containerPort, with server listening on hostPort.
		// Exposed ports for router should be hostPort:hostPort so router
		// can determine on which port a request came in
		// and route the request to the correct upstream
		exposePort := ""
		var ports []string

		// Each port pair should be of the form <number>:<number> or <number>
		// It's possible to have received a malformed HTTP_EXPOSE or HTTPS_EXPOSE from
		// some random container, so don't break if that happens.
		if !regexp.MustCompile(`^[0-9]+(:[0-9]+)?$`).MatchString(exposePortPair) {
			continue
		}

		if strings.Contains(exposePortPair, ":") {
			ports = strings.Split(exposePortPair, ":")
		} else {
			// HTTP_EXPOSE and HTTPS_EXPOSE can be a single port, meaning port:port
			ports = []string{exposePortPair, exposePortPair}
		}
		exposePort = ports[0]

		var match bool
		for _, routerPort := range routerPorts {
			if exposePort == routerPort {
				match = true
			}
		}

		// If no match, we are adding a new port mapping
		if !match {
			routerPorts = append(routerPorts, exposePort)
		}
	}

	return routerPorts
}

// CheckRouterPorts tries to connect to the ports the router will use as a heuristic to find out
// if they're available for docker to bind to. Returns an error if either one results
// in a successful connection.
func CheckRouterPorts(activeApps []*DdevApp) error {
	routerContainer, _ := FindDdevRouter()
	var existingExposedPorts []string
	var err error
	if routerContainer != nil {
		existingExposedPorts, err = dockerutil.GetBoundHostPorts(routerContainer.ID)
		if err != nil {
			return err
		}
	}
	newRouterPorts := determineRouterPorts(activeApps)

	for _, port := range newRouterPorts {
		if nodeps.ArrayContainsString(existingExposedPorts, port) {
			continue
		}
		if netutil.IsPortActive(port) {
			return fmt.Errorf("port %s is already in use", port)
		}
	}
	return nil
}

// AllocateAvailablePortForRouter finds an available port in the local machine, in the range provided.
// Returns the port found, and a boolean that determines if the
// port is valid (true) or not (false), and the port is marked as allocated
func AllocateAvailablePortForRouter(start, upTo int) (int, bool) {
	for p := start; p <= upTo; p++ {
		// If we have already assigned this port, continue looking
		if _, portAlreadyUsed := EphemeralRouterPortsAssigned[p]; portAlreadyUsed {
			continue
		}
		// But if we find the port is still available, use it, after marking it as assigned
		if !netutil.IsPortActive(fmt.Sprint(p)) {
			EphemeralRouterPortsAssigned[p] = true
			return p, true
		}
	}

	return 0, false
}

// GetAvailableRouterPort gets an ephemeral replacement port when the
// proposedPort is not available.
//
// The function returns an ephemeral port if the proposedPort is bound by a process
// in the host other than the running router.
//
// Returns the original proposedPort, the ephemeral port found,
// and a bool which is true if the proposedPort has been
// replaced with an ephemeralPort
func GetAvailableRouterPort(proposedPort string, minPort, maxPort int) (string, string, bool) {
	// If the proposedPort is empty, we don't need to do anything
	if proposedPort == "" {
		return proposedPort, "", false
	}
	// If the router is alive and well, we can see if it's already handling the proposedPort
	status, _ := GetRouterStatus()
	if status == "healthy" {
		util.Debug("GetAvailableRouterPort(): Router is healthy and running")
		r, err := FindDdevRouter()
		// If we have error getting router (Impossible, because we just got healthy status)
		if err != nil {
			return proposedPort, "", false
		}

		// Check if the proposedPort is already being handled by the router.
		routerPortsAlreadyBound, err := dockerutil.GetBoundHostPorts(r.ID)
		if err != nil {
			// If error getting ports (mostly impossible)
			return proposedPort, "", false
		}
		if nodeps.ArrayContainsString(routerPortsAlreadyBound, proposedPort) {
			// If the proposedPort is already bound by the router,
			// there's no need to go find an ephemeral port.
			util.Debug("GetAvailableRouterPort(): proposedPort %s already bound on ddev-router, accepting it", proposedPort)
			return proposedPort, "", false
		}
	}

	// At this point, the router may or may not be running, but we
	// have not found it already having the proposedPort bound
	if !netutil.IsPortActive(proposedPort) {
		// If the proposedPort is available (not active) for use, just have the router use it
		util.Debug("GetAvailableRouterPort(): proposedPort %s is available, use proposedPort=%s", proposedPort, proposedPort)
		return proposedPort, "", false
	}

	ephemeralPort, ok := AllocateAvailablePortForRouter(minPort, maxPort)
	if !ok {
		// Unlikely
		util.Debug("GetAvailableRouterPort(): unable to AllocateAvailablePortForRouter()")
		return proposedPort, "", false
	}

	util.Debug("GetAvailableRouterPort(): proposedPort %s is not available, epheneralPort=%d is available, use it", proposedPort, ephemeralPort)

	return proposedPort, strconv.Itoa(ephemeralPort), true
}

// GetEphemeralPortsIfNeeded replaces the provided ports with an ephemeral version if they need it.
func GetEphemeralPortsIfNeeded(ports []*string, verbose bool) {
	for _, port := range ports {
		proposedPort, replacementPort, portChangeRequired := GetAvailableRouterPort(*port, MinEphemeralPort, MaxEphemeralPort)
		if portChangeRequired {
			*port = replacementPort
			if verbose {
				output.UserOut.Printf("Port %s is busy, using %s instead, see %s", proposedPort, replacementPort, "https://ddev.com/s/port-conflict")
			}
		}
	}
}

// AssignRouterPortsToGenericWebserverPorts assigns the router ports to the generic webserver.
// If it's a generic webserver, use the first pair of exposed ports as router ports.
func AssignRouterPortsToGenericWebserverPorts(app *DdevApp) {
	if app.WebserverType == nodeps.WebserverGeneric && len(app.WebExtraExposedPorts) > 0 {
		app.RouterHTTPPort = strconv.Itoa(app.WebExtraExposedPorts[0].HTTPPort)
		app.RouterHTTPSPort = strconv.Itoa(app.WebExtraExposedPorts[0].HTTPSPort)
	}
}

// SyncGenericWebserverPortsWithRouterPorts syncs the generic webserver ports with the router ports.
// If the used ephemeral router ports are different, the first pair webserver ports should be updated.
func SyncGenericWebserverPortsWithRouterPorts(app *DdevApp) {
	if app.WebserverType == nodeps.WebserverGeneric && len(app.WebExtraExposedPorts) > 0 {
		if httpPort, err := strconv.Atoi(app.GetPrimaryRouterHTTPPort()); err == nil {
			app.WebExtraExposedPorts[0].HTTPPort = httpPort
		}
		if httpsPort, err := strconv.Atoi(app.GetPrimaryRouterHTTPSPort()); err == nil {
			app.WebExtraExposedPorts[0].HTTPSPort = httpsPort
		}
	}
}
