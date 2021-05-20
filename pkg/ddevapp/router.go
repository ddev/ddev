package ddevapp

import (
	"bytes"
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/netutil"
	"github.com/drud/ddev/pkg/nodeps"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
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

// StopRouterIfNoContainers stops the router if there are no ddev containers running.
func StopRouterIfNoContainers() error {
	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		err = dockerutil.RemoveContainer(nodeps.RouterContainer, 0)
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
	// If the router is not healthy/running, we'll kill it so it
	// starts over again.
	router, err := FindDdevRouter()
	if router != nil && err == nil && router.State != "running" {
		err = dockerutil.RemoveContainer(nodeps.RouterContainer, 0)
		if err != nil {
			return err
		}
	}

	routerComposeFullPath, err := generateRouterCompose()
	if err != nil {
		return err
	}
	err = CheckRouterPorts()
	if err != nil {
		return fmt.Errorf("Unable to listen on required ports, %v,\nTroubleshooting suggestions at https://ddev.readthedocs.io/en/stable/users/troubleshooting/#unable-listen", err)
	}

	// run docker-compose up -d against the ddev-router full compose file
	_, _, err = dockerutil.ComposeCmd([]string{routerComposeFullPath}, "-p", RouterProjectName, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	// ensure we have a happy router
	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	logOutput, err := dockerutil.ContainerWait(containerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready; debug with 'docker logs ddev-router'; logOutput=%s, err=%v", logOutput, err)
	}

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

	templ := template.New("routerTemplate")
	templ, err := templ.Parse(DdevRouterTemplate)
	if err != nil {
		return "", err
	}

	dockerIP, _ := dockerutil.GetDockerIP()

	templateVars := map[string]interface{}{
		"router_image":               version.RouterImage,
		"router_tag":                 version.RouterTag,
		"ports":                      exposedPorts,
		"router_bind_all_interfaces": globalconfig.DdevGlobalConfig.RouterBindAllInterfaces,
		"compose_version":            version.DockerComposeFileFormatVersion,
		"dockerIP":                   dockerIP,
		"disable_http2":              globalconfig.DdevGlobalConfig.DisableHTTP2,
		"letsencrypt":                globalconfig.DdevGlobalConfig.UseLetsEncrypt,
		"letsencrypt_email":          globalconfig.DdevGlobalConfig.LetsEncryptEmail,
		"AutoRestartContainers":      globalconfig.DdevGlobalConfig.AutoRestartContainers,
	}

	err = templ.Execute(&doc, templateVars)
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

// FindDdevRouter usees FindContainerByLabels to get our router container and
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
	status, logOutput := GetRouterStatus()
	var renderedStatus string
	badRouter := "\nThe router is not yet healthy. Your projects may not be accessible.\nIf it doesn't become healthy try running 'ddev start' on a project to recreate it."

	switch status {
	case SiteStopped:
		renderedStatus = color.RedString(status) + badRouter
	case "healthy":
		renderedStatus = color.CyanString(status)
	case "exited":
		fallthrough
	default:
		renderedStatus = color.RedString(status) + badRouter + "\n" + logOutput
	}
	return fmt.Sprintf("\nDDEV ROUTER STATUS: %v", renderedStatus)
}

// GetRouterStatus retur s router status and warning if not
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

			for _, exposePort := range exposePorts {
				// ports defined as hostPort:containerPort allow for router to configure upstreams
				// for containerPort, with server listening on hostPort. exposed ports for router
				// should be hostPort:hostPort so router can determine what port a request came from
				// and route the request to the correct upstream
				if strings.Contains(exposePort, ":") {
					ports := strings.Split(exposePort, ":")
					exposePort = ports[0]
				}

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
