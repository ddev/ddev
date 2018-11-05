package ddevapp

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"

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
	ddevDir := util.GetGlobalDdevDir()
	dest := path.Join(ddevDir, "router-compose.yaml")
	return dest
}

// StopRouterIfNoContainers stops the router if there are no ddev containers running.
func StopRouterIfNoContainers() error {

	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		dest := RouterComposeYAMLPath()
		_, _, err = dockerutil.ComposeCmd([]string{dest}, "-p", RouterProjectName, "down")
		return err
	}
	return nil
}

// StartDdevRouter ensures the router is running.
func StartDdevRouter() error {
	exposedPorts := determineRouterPorts()

	routerComposePath := RouterComposeYAMLPath()
	routerdir := filepath.Dir(routerComposePath)
	err := os.MkdirAll(routerdir, 0755)
	if err != nil {
		return fmt.Errorf("unable to create directory for ddev router: %v", err)
	}

	certDir := filepath.Join(util.GetGlobalDdevDir(), "certs")
	if _, err = os.Stat(certDir); os.IsNotExist(err) {
		err = os.MkdirAll(certDir, 0755)
		if err != nil {
			return fmt.Errorf("unable to create directory for ddev certs: %v", err)
		}
	}

	var doc bytes.Buffer
	f, ferr := os.Create(routerComposePath)
	if ferr != nil {
		return ferr
	}
	defer util.CheckClose(f)

	templ := template.New("compose template")
	templ, err = templ.Parse(DdevRouterTemplate)
	if err != nil {
		return err
	}

	templateVars := map[string]interface{}{
		"router_image":    version.RouterImage,
		"router_tag":      version.RouterTag,
		"ports":           exposedPorts,
		"compose_version": version.DockerComposeFileFormatVersion,
	}

	err = templ.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)

	// Since the ports in use may have changed, stop the router and make sure
	// we can access all ports.
	// It might be possible to do this instead by reading from docker all the
	// existing mapped ports.
	_, _, err = dockerutil.ComposeCmd([]string{routerComposePath}, "-p", RouterProjectName, "down")
	util.CheckErr(err)

	err = CheckRouterPorts()
	if err != nil {
		return fmt.Errorf("Unable to listen on required ports, %v,\nTroubleshooting suggestions at https://ddev.readthedocs.io/en/latest/users/troubleshooting/#unable-listen", err)
	}

	// run docker-compose up -d in the newly created directory
	_, _, err = dockerutil.ComposeCmd([]string{routerComposePath}, "-p", RouterProjectName, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	// ensure we have a happy router
	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	log, err := dockerutil.ContainerWait(containerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready: log=%s, err=%v", log, err)
	}

	return nil
}

// findDdevRouter usees FindContainerByLabels to get our router container and
// return it. This is currently unused but may be useful in the future.
// nolint: deadcode
func findDdevRouter() (*docker.APIContainers, error) {
	containerQuery := map[string]string{
		"com.docker.compose.service": RouterProjectName,
	}
	container, err := dockerutil.FindContainerByLabels(containerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findContainersByLabels, %v", err)
	}
	return container, nil
}

// RenderRouterStatus returns a user-friendly string showing router-status
func RenderRouterStatus() string {
	status, log := GetRouterStatus()
	var renderedStatus string
	badRouter := "\nThe router is not currently healthy. Your projects may not be inaccessible.\nTry running 'ddev start' on a site to recreate the router."

	switch status {
	case SiteNotFound:
		renderedStatus = color.RedString(status) + badRouter
	case "healthy":
		renderedStatus = color.CyanString(status)
	case "exited":
		fallthrough
	default:
		renderedStatus = color.RedString(status) + badRouter + ":" + log
	}
	return fmt.Sprintf("\nDDEV ROUTER STATUS: %v", renderedStatus)
}

// GetRouterStatus retur s router status and warning if not
// running or healthy, as applicable.
// return status and most recent log
func GetRouterStatus() (string, string) {
	var status, log string

	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	container, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		status = SiteNotFound
	} else {
		status, log = dockerutil.GetContainerHealth(container)
	}

	return status, log
}

// determineRouterPorts returns a list of port mappings retrieved from running site
// containers defining VIRTUAL_PORT env var
func determineRouterPorts() []string {
	var routerPorts []string
	containers, err := dockerutil.GetDockerContainers(false)
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
	return routerPorts
}

// CheckRouterPorts tries to connect to the ports the router will use as a heuristic to find out
// if they're available for docker to bind to. Returns an error if either one results
// in a successful connection.
func CheckRouterPorts() error {
	routerPorts := determineRouterPorts()
	for _, port := range routerPorts {
		if util.IsPortActive(port) {
			return fmt.Errorf("localhost port %s is in use", port)
		}
	}
	return nil
}
