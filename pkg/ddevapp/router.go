package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

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
	if nodeps.IsCodespaces() || nodeps.IsDevcontainer() {
		return true
	}
	return nodeps.ArrayContainsString(app.GetOmittedContainers(), globalconfig.DdevRouterContainer)
}

// RemoveRouterContainer stops and removes the ddev-router container.
func RemoveRouterContainer() error {
	_, err := FindDdevRouter()
	if err != nil {
		// Router not found, nothing to remove
		return nil
	}
	err = dockerutil.RemoveContainer(nodeps.RouterContainer)
	if err != nil {
		if ok := dockerutil.IsErrNotFound(err); !ok {
			return err
		}
	}
	return nil
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
		router = nil
	}

	activeApps := GetActiveProjects()

	// Check if router needs to be recreated due to port changes
	needsRecreation := false
	if router != nil && err == nil && router.State == "running" {
		// Router is running, check if ports have changed
		existingPorts, err := dockerutil.GetBoundHostPorts(router.ID)
		if err != nil {
			util.Debug("Error getting bound ports, will recreate router: %v", err)
			needsRecreation = true
		} else {
			neededPorts := determineRouterPorts(activeApps)
			// Add the Traefik monitor port to the needed list for comparison
			// (it's always bound by the router but not returned by determineRouterPorts
			// since it's added separately in the static config template)
			neededPorts = append(neededPorts, globalconfig.DdevGlobalConfig.TraefikMonitorPort)
			util.Debug("Router port comparison: existing=%v needed=%v match=%v", existingPorts, neededPorts, PortsMatch(existingPorts, neededPorts))
			if !PortsMatch(existingPorts, neededPorts) {
				util.Debug("Router ports have changed, will recreate router")
				needsRecreation = true
			} else {
				util.Debug("Router ports have not changed, skipping recreation")
			}
		}
	} else {
		// Router is not running, needs to be started
		needsRecreation = true
	}

	if needsRecreation {
		output.UserOut.Printf("Starting %s, pushing config...", nodeps.RouterContainer)
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
			return fmt.Errorf("unable to listen on required ports, %v\nTroubleshooting suggestions at https://docs.ddev.com/en/stable/users/usage/troubleshooting/#unable-listen", err)
		}

		// Run docker-compose up -d against the ddev-router full compose file
		_, _, err = dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{routerComposeFullPath},
			Action:       []string{"-p", RouterComposeProjectName, "up", "--build", "-d"},
		})
		if err != nil {
			return fmt.Errorf("failed to start ddev-router: %v", err)
		}
	} else {
		output.UserOut.Printf("%s already running, pushing new config...", nodeps.RouterContainer)

		// Even if we don't recreate, update the Traefik config for the new project
		err = PushGlobalTraefikConfig(activeApps)
		if err != nil {
			return fmt.Errorf("failed to push global Traefik config: %v", err)
		}

		// Force the healthcheck to run and wait for Traefik to load the new config.
		err = ClearRouterHealthcheck()
		if err != nil {
			return err
		}
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
	wait := output.StartWait(fmt.Sprintf("Waiting for %s to become ready", nodeps.RouterContainer))
	util.Debug("Router wait: checking for container with labels %v, polling every 500ms for healthy status", label)
	logOutput, err := dockerutil.ContainerWait(routerWaitTimeout, label)
	elapsed := wait.Complete(err)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready after %.1fs; log=%s, err=%v", elapsed.Seconds(), logOutput, err)
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
	// On remote Docker hosts, the Docker IP (e.g. a cloud provider's public IP)
	// is not a valid bind address on the Docker host itself, so bind to all interfaces.
	if dockerutil.IsRemoteDockerHost() {
		dockerIP = "0.0.0.0"
	}

	uid, gid, username := dockerutil.GetContainerUser()
	timezone, _ := util.GetLocalTimezone()

	templateVars := map[string]interface{}{
		"Username":                   username,
		"UID":                        uid,
		"GID":                        gid,
		"router_image":               ddevImages.GetRouterImage(),
		"ports":                      exposedPorts,
		"router_bind_all_interfaces": globalconfig.DdevGlobalConfig.RouterBindAllInterfaces || dockerutil.IsRemoteDockerHost(),
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

// PortsMatch returns true if the existing ports contain all the needed ports.
// It's fine for the router to have extra ports bound (from other projects that have stopped),
// we only need to recreate the router when it's missing ports we need.
func PortsMatch(existingPorts, neededPorts []string) bool {
	// Create a map of existing ports for quick lookup
	existingMap := make(map[string]bool)
	for _, port := range existingPorts {
		existingMap[port] = true
	}

	// Check if all needed ports are in existing ports
	for _, port := range neededPorts {
		if !existingMap[port] {
			return false
		}
	}

	return true
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
				errorInfo = fmt.Sprintf("Detected configuration error(s):\n%s", configErrors)
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

// ClearRouterHealthcheck forces the router healthcheck to run immediately by
// removing the healthy marker and error file, then executing the healthcheck.
// This ensures the router status reflects the current configuration state.
func ClearRouterHealthcheck() error {
	router, err := FindDdevRouter()
	if err != nil || router == nil {
		// Router not found or error - nothing to clear
		return nil
	}

	util.Debug("Forcing router healthcheck to clear status")
	uid, _, _ := dockerutil.GetContainerUser()
	_, _, err = dockerutil.Exec(router.ID, "rm -f /tmp/healthy /tmp/ddev-traefik-errors.txt && /healthcheck.sh", uid)
	if err != nil {
		return fmt.Errorf("router healthcheck failed: %v", err)
	}
	return nil
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

	// Check if any of the new ports are already in use
	var portError error
	for _, port := range newRouterPorts {
		if nodeps.ArrayContainsString(existingExposedPorts, port) {
			continue
		}
		if netutil.IsPortActive(port) {
			portError = fmt.Errorf("port %s is already in use", port)
			break
		}
	}

	// If we found a port conflict, check if it might be a security software false positive
	if portError != nil {
		// If all ephemeral ports appear active, it's likely security software interference.
		// Let Docker report any real conflicts.
		// See https://github.com/ddev/ddev/issues/7921
		freePortsAvailable := false
		for p := MinEphemeralPort; p <= MaxEphemeralPort; p++ {
			if !netutil.IsPortActive(fmt.Sprint(p)) {
				freePortsAvailable = true
				break
			}
		}
		if !freePortsAvailable {
			util.WarningOnce("Unable to check port availability")
			util.WarningOnce("Assuming ports are available, see https://ddev.com/s/port-conflict")
			return nil
		}
		// There are free ports available, so this is a real conflict
		return portError
	}

	return nil
}

// AllocateAvailablePortForRouter finds an available port in the local machine, in the range provided.
// Returns the port found, and a boolean that determines if the
// port is valid (true) or not (false), and the port is marked as allocated
func AllocateAvailablePortForRouter(start, upTo int) (int, bool) {
	// Get ports already bound by the router - these can be reused
	var routerBoundPorts []string
	if router, err := FindDdevRouter(); err == nil && router != nil {
		routerBoundPorts, _ = dockerutil.GetBoundHostPorts(router.ID)
	}

	for p := start; p <= upTo; p++ {
		portStr := fmt.Sprint(p)
		// If we have already assigned this port in this session, continue looking
		if _, portAlreadyUsed := EphemeralRouterPortsAssigned[p]; portAlreadyUsed {
			continue
		}
		// If the port is already bound by the router, we can reuse it
		if nodeps.ArrayContainsString(routerBoundPorts, portStr) {
			util.Debug("AllocateAvailablePortForRouter: port %s is already bound by router, reusing it", portStr)
			EphemeralRouterPortsAssigned[p] = true
			return p, true
		}
		// If the port is not active (available), use it
		if !netutil.IsPortActive(portStr) {
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
	// If the router exists, check if it's already handling the proposedPort
	// regardless of its health status. This prevents allocating ephemeral ports
	// when the router is running but unhealthy (e.g., broken Traefik config).
	r, err := FindDdevRouter()
	if r != nil && err == nil {
		util.Debug("GetAvailableRouterPort(): Router exists, checking bound ports")
		// Check if the proposedPort is already being handled by the router.
		routerPortsAlreadyBound, err := dockerutil.GetBoundHostPorts(r.ID)
		if err != nil {
			util.Debug("GetAvailableRouterPort(): Error getting bound ports: %v", err)
			// Continue to port availability check below
		} else if nodeps.ArrayContainsString(routerPortsAlreadyBound, proposedPort) {
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
		// Unlikely, but this can happen if security software makes all ports appear active.
		util.Debug("GetAvailableRouterPort(): proposedPort %s is not available, no ephemeral ports in range %d-%d are available", proposedPort, minPort, maxPort)
		return proposedPort, "", false
	}

	util.Debug("GetAvailableRouterPort(): proposedPort %s is not available, ephemeralPort=%d is available, use it", proposedPort, ephemeralPort)

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
// For generic webservers, the router ports come from WebExtraExposedPorts[0].
//
// IMPORTANT: This function assumes SortWebExtraExposedPorts was called by app.ReadConfig,
// which ensures the entry matching the configured router ports (80/443 by default) is at index 0.
// See SortWebExtraExposedPorts for the sorting logic.
func AssignRouterPortsToGenericWebserverPorts(app *DdevApp) {
	if app.WebserverType == nodeps.WebserverGeneric && len(app.WebExtraExposedPorts) > 0 {
		app.RouterHTTPPort = strconv.Itoa(app.WebExtraExposedPorts[0].HTTPPort)
		app.RouterHTTPSPort = strconv.Itoa(app.WebExtraExposedPorts[0].HTTPSPort)
	}
}

// SyncGenericWebserverPortsWithRouterPorts updates WebExtraExposedPorts[0] with ephemeral ports.
// When configured ports (e.g., 80/443) are busy, DDEV assigns ephemeral ports instead.
// This function syncs those ephemeral ports back to the primary WebExtraExposedPorts entry.
//
// IMPORTANT: This function assumes SortWebExtraExposedPorts was called by app.ReadConfig,
// which ensures the primary entry is at index 0. See SortWebExtraExposedPorts for details.
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

// SortWebExtraExposedPorts sorts WebExtraExposedPorts so the entry matching
// the configured router ports comes first (index 0).
//
// This is called by app.ReadConfig to ensure AssignRouterPortsToGenericWebserverPorts
// and SyncGenericWebserverPortsWithRouterPorts work correctly with WebExtraExposedPorts[0].
//
// Priority for matching: app.RouterHTTPPort -> global config -> defaults (80/443).
// Entries with full match (both HTTP and HTTPS) come before partial matches.
func SortWebExtraExposedPorts(app *DdevApp) {
	if len(app.WebExtraExposedPorts) <= 1 {
		return
	}
	// Priority: app config -> global config -> defaults
	preferredHTTP := app.RouterHTTPPort
	if preferredHTTP == "" {
		preferredHTTP = globalconfig.DdevGlobalConfig.RouterHTTPPort
	}
	if preferredHTTP == "" {
		preferredHTTP = nodeps.DdevDefaultRouterHTTPPort
	}
	preferredHTTPS := app.RouterHTTPSPort
	if preferredHTTPS == "" {
		preferredHTTPS = globalconfig.DdevGlobalConfig.RouterHTTPSPort
	}
	if preferredHTTPS == "" {
		preferredHTTPS = nodeps.DdevDefaultRouterHTTPSPort
	}
	httpPort, _ := strconv.Atoi(preferredHTTP)
	httpsPort, _ := strconv.Atoi(preferredHTTPS)

	// Sort ports so the best match for the requested HTTP/HTTPS ports comes first.
	// The order is stable, so ports with the same match quality keep their
	// original relative order.
	slices.SortStableFunc(app.WebExtraExposedPorts, func(a, b WebExposedPort) int {
		aMatch := 0
		bMatch := 0

		// Match scoring:
		// 2 = both HTTP and HTTPS ports match exactly
		// 1 = either HTTP or HTTPS port matches
		// 0 = no match at all
		if a.HTTPPort == httpPort && a.HTTPSPort == httpsPort {
			aMatch = 2
		} else if a.HTTPPort == httpPort || a.HTTPSPort == httpsPort {
			aMatch = 1
		}
		if b.HTTPPort == httpPort && b.HTTPSPort == httpsPort {
			bMatch = 2
		} else if b.HTTPPort == httpPort || b.HTTPSPort == httpsPort {
			bMatch = 1
		}
		// Sort in descending order so higher-quality matches appear first
		return bMatch - aMatch
	})
}
