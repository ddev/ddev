package platform

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"

	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fatih/color"

	homedir "github.com/mitchellh/go-homedir"
)

const routerProjectName = "ddev-router"

// RouterComposeYAMLPath returns the full filepath to the routers docker-compose yaml file.
func RouterComposeYAMLPath() string {
	userHome, err := homedir.Dir()
	if err != nil {
		log.Fatal("could not get home directory for current user. is it set?")
	}
	routerdir := path.Join(userHome, ".ddev")
	dest := path.Join(routerdir, "router-compose.yaml")

	return dest
}

// StopRouter stops the local router if there are no ddev containers running.
func StopRouter() error {

	containersRunning, err := ddevContainersRunning()
	if err != nil {
		return err
	}

	if !containersRunning {
		dest := RouterComposeYAMLPath()
		return dockerutil.ComposeCmd([]string{dest}, "-p", routerProjectName, "down", "-v")
	}
	return nil
}

// StartDdevRouter ensures the router is running.
func StartDdevRouter() error {
	exposedPorts := determineRouterPorts()

	dest := RouterComposeYAMLPath()
	routerdir := filepath.Dir(dest)
	err := os.MkdirAll(routerdir, 0755)
	if err != nil {
		return fmt.Errorf("unable to create directory for ddev router: %s", err)
	}

	var doc bytes.Buffer
	f, ferr := os.Create(dest)
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
		"router_image": version.RouterImage,
		"router_tag":   version.RouterTag,
		"ports":        exposedPorts,
	}

	err = templ.Execute(&doc, templateVars)
	util.CheckErr(err)
	_, err = f.WriteString(doc.String())
	util.CheckErr(err)

	// run docker-compose up -d in the newly created directory
	err = dockerutil.ComposeCmd([]string{dest}, "-p", routerProjectName, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	fmt.Println("Starting service health checks...")

	// ensure we have a happy router
	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	err = dockerutil.ContainerWait(35, label)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready: %v", err)
	}

	return nil
}

// PrintRouterStatus outputs router status and warning if not
// running or healthy, as applicable.
func PrintRouterStatus() string {
	var status string

	badRouter := "\nThe router is not currently running. Your sites are likely inaccessible at this time.\nTry running 'ddev start' on a site to recreate the router."

	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	container, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		status = color.RedString("not running") + badRouter
	}

	status = dockerutil.GetContainerHealth(container)

	switch status {
	case "exited":
		status = color.YellowString(SiteStopped) + badRouter
	case "restarting":
		status = color.RedString(status) + badRouter
	case "healthy":
		status = color.CyanString(SiteRunning)
	default:
		status = color.CyanString(status)
	}

	return fmt.Sprintf("\nDDEV ROUTER STATUS: %v", status)
}

// determineRouterPorts returns a list of port mappings retrieved from running site
// containers defining VIRTUAL_PORT env var
func determineRouterPorts() []string {
	var routerPorts []string
	containers, err := dockerutil.GetDockerContainers(false)
	if err != nil {
		log.Fatal("failed to retreive containers for determining port mappings", err)
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
