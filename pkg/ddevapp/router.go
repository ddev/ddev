package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	dockerImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
)

// RouterProjectName is the "machine name" of the router docker-compose
const RouterProjectName = "ddev-router"

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
	if nodeps.IsGitpod() || nodeps.IsCodespaces() {
		return true
	}
	return nodeps.ArrayContainsString(app.GetOmittedContainers(), globalconfig.DdevRouterContainer)
}

// StopRouterIfNoContainers stops the router if there are no ddev containers running.
func StopRouterIfNoContainers() error {
	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		err = dockerutil.RemoveContainer(nodeps.RouterContainer)
		if err != nil {
			if _, ok := err.(*docker.NoSuchContainer); !ok {
				return err
			}
		}
	}
	return nil
}

// StartDdevRouter ensures the router is running.
func StartDdevRouter() error {
	err := os.MkdirAll(filepath.Join(globalconfig.GetGlobalDdevDir(), "router-build"), 0755)
	if err != nil {
		return err
	}
	// If the router is not healthy/running, we'll kill it so it
	// starts over again.
	router, err := FindDdevRouter()
	if router != nil && err == nil && router.State != "running" {
		err = dockerutil.RemoveContainer(nodeps.RouterContainer)
		if err != nil {
			return err
		}
	}

	routerComposeFullPath, err := generateRouterCompose()
	if err != nil {
		return err
	}
	err = GenerateRouterDockerfile()
	if err != nil {
		return err
	}
	if globalconfig.DdevGlobalConfig.IsTraefikRouter() {
		err = fileutil.CopyEmbedAssets(bundledAssets, "healthcheck/router/traefik", filepath.Join(globalconfig.GetGlobalDdevDir(), "router-build"))
		if err != nil {
			return fmt.Errorf("failed to save traefik healthcheck: %v", err)
		}
		err = pushGlobalTraefikConfig()
		if err != nil {
			return fmt.Errorf("failed to push global traefik config: %v", err)
		}
	}

	err = CheckRouterPorts()
	if err != nil {
		return fmt.Errorf("Unable to listen on required ports, %v,\nTroubleshooting suggestions at https://ddev.readthedocs.io/en/stable/users/basics/troubleshooting/#unable-listen", err)
	}

	// run docker-compose up -d against the ddev-router full compose file
	_, _, err = dockerutil.ComposeCmd([]string{routerComposeFullPath}, "-p", RouterProjectName, "up", "--build", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	// ensure we have a happy router
	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	// Normally the router comes right up, but when
	// it has to do let's encrypt updates, it can take
	// some time.
	routerWaitTimeout := 60
	if globalconfig.DdevGlobalConfig.UseLetsEncrypt {
		routerWaitTimeout = 180
	}
	util.Debug(`Waiting for ddev-router to become ready. docker inspect --format "{{json .State.Health }}" ddev-router`)
	logOutput, err := dockerutil.ContainerWait(routerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready; debug with 'docker logs ddev-router' and 'docker inspect --format \"{{json .State.Health }}\" ddev-router'; logOutput=%s, err=%v", logOutput, err)
	}
	util.Debug("ddev-router is ready")

	return nil
}

// generateRouterCompose() generates the ~/.ddev/.router-compose.yaml and ~/.ddev/.router-compose-full.yaml
func generateRouterCompose() (string, error) {
	exposedPorts := determineRouterPorts()

	routerComposeBasePath := RouterComposeYAMLPath()
	routerComposeFullPath := FullRenderedRouterComposeYAMLPath()

	var doc bytes.Buffer
	f, ferr := os.Create(routerComposeBasePath)
	if ferr != nil {
		return "", ferr
	}
	defer util.CheckClose(f)

	dockerIP, _ := dockerutil.GetDockerIP()

	uid, gid, username := util.GetContainerUIDGid()

	templateVars := map[string]interface{}{
		"Username":                   username,
		"UID":                        uid,
		"GID":                        gid,
		"router_image":               dockerImages.GetRouterImage(),
		"ports":                      exposedPorts,
		"router_bind_all_interfaces": globalconfig.DdevGlobalConfig.RouterBindAllInterfaces,
		"dockerIP":                   dockerIP,
		"disable_http2":              globalconfig.DdevGlobalConfig.DisableHTTP2,
		"letsencrypt":                globalconfig.DdevGlobalConfig.UseLetsEncrypt,
		"letsencrypt_email":          globalconfig.DdevGlobalConfig.LetsEncryptEmail,
		"AutoRestartContainers":      globalconfig.DdevGlobalConfig.AutoRestartContainers,
		"Router":                     globalconfig.DdevGlobalConfig.Router,
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
	fullContents, _, err := dockerutil.ComposeCmd(files, "config")
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
func FindDdevRouter() (*docker.APIContainers, error) {
	containerQuery := map[string]string{
		"com.docker.compose.service": RouterProjectName,
	}
	container, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	if container == nil {
		return nil, fmt.Errorf("No ddev-router was found")
	}
	return container, nil
}

// RenderRouterStatus returns a user-friendly string showing router-status
func RenderRouterStatus() string {
	var renderedStatus string
	if !nodeps.ArrayContainsString(globalconfig.DdevGlobalConfig.OmitContainersGlobal, globalconfig.DdevRouterContainer) {
		status, logOutput := GetRouterStatus()
		badRouter := "The router is not healthy. Your projects may not be accessible.\nIf it doesn't become healthy try running 'ddev poweroff && ddev start' on a project to recreate it."

		switch status {
		case SiteStopped:
			renderedStatus = util.ColorizeText(status, "red") + " " + badRouter
		case "healthy":
			renderedStatus = util.ColorizeText(status, "green")
		case "exited":
			fallthrough
		default:
			renderedStatus = util.ColorizeText(status, "red") + " " + badRouter + "\n" + logOutput
		}
	}
	return renderedStatus
}

// GetRouterStatus returns router status and warning if not
// running or healthy, as applicable.
// return status and most recent log
func GetRouterStatus() (string, string) {
	var status, logOutput string
	container, err := FindDdevRouter()

	if err != nil || container == nil {
		status = SiteStopped
	} else {
		status, logOutput = dockerutil.GetContainerHealth(container)
	}

	return status, logOutput
}

// determineRouterPorts returns a list of port mappings retrieved from running site
// containers defining VIRTUAL_PORT env var
func determineRouterPorts() []string {
	var routerPorts []string
	containers, err := dockerutil.FindContainersWithLabel("com.ddev.site-name")
	if err != nil {
		util.Failed("failed to retrieve containers for determining port mappings: %v", err)
	}

	// loop through all containers with site-name label
	for _, container := range containers {
		if _, ok := container.Labels["com.ddev.site-name"]; ok {
			if container.State != "running" {
				continue
			}
			var exposePorts []string

			httpPorts := dockerutil.GetContainerEnv("HTTP_EXPOSE", container)
			if httpPorts != "" {
				ports := strings.Split(httpPorts, ",")
				exposePorts = append(exposePorts, ports...)
			}

			httpsPorts := dockerutil.GetContainerEnv("HTTPS_EXPOSE", container)
			if httpsPorts != "" {
				ports := strings.Split(httpsPorts, ",")
				exposePorts = append(exposePorts, ports...)
			}

			for _, exposePortPair := range exposePorts {
				// ports defined as hostPort:containerPort allow for router to configure upstreams
				// for containerPort, with server listening on hostPort. exposed ports for router
				// should be hostPort:hostPort so router can determine what port a request came from
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

				// if no match, we are adding a new port mapping
				if !match {
					routerPorts = append(routerPorts, exposePort)
				}
			}
		}
	}
	sort.Slice(routerPorts, func(i, j int) bool {
		return routerPorts[i] < routerPorts[j]
	})

	return routerPorts
}

// CheckRouterPorts tries to connect to the ports the router will use as a heuristic to find out
// if they're available for docker to bind to. Returns an error if either one results
// in a successful connection.
func CheckRouterPorts() error {
	routerContainer, _ := FindDdevRouter()
	var existingExposedPorts []string
	var err error
	if routerContainer != nil {
		existingExposedPorts, err = dockerutil.GetExposedContainerPorts(routerContainer.ID)
		if err != nil {
			return err
		}
	}
	newRouterPorts := determineRouterPorts()

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

func GenerateRouterDockerfile() error {

	type routerData struct {
		Router string
	}
	templateData := routerData{
		Router: globalconfig.DdevGlobalConfig.Router,
	}

	routerDockerfile := filepath.Join(globalconfig.GetGlobalDdevDir(), "router-build", "Dockerfile")
	sigExists := true
	//TODO: Systematize this checking-for-signature, allow an arg to skip if empty
	fi, err := os.Stat(routerDockerfile)
	// Don't use simple fileutil.FileExists() because of the danger of an empty file
	if err == nil && fi.Size() > 0 {
		// Check to see if file has #ddev-generated in it, meaning we can recreate it.
		sigExists, err = fileutil.FgrepStringInFile(routerDockerfile, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}
	}
	if !sigExists {
		util.Debug("Not creating %s because it exists and is managed by user", routerDockerfile)
	} else {
		f, err := os.Create(routerDockerfile)
		if err != nil {
			util.Failed("failed to create router Dockerfile: %v", err)
		}
		t, err := template.New("router_Dockerfile_template").Funcs(sprig.TxtFuncMap()).ParseFS(bundledAssets, "router_Dockerfile_template")
		if err != nil {
			return fmt.Errorf("could not create template from router_Dockerfile_template: %v", err)
		}

		err = t.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("could not parse router_Dockerfile_template with templatedate='%v':: %v", templateData, err)
		}
	}
	return nil
}
