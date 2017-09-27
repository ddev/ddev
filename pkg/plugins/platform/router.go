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

	"net"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/fatih/color"
)

const RouterProjectName = "ddev-router"

// RouterComposeYAMLPath returns the full filepath to the routers docker-compose yaml file.
func RouterComposeYAMLPath() string {
	ddevDir := util.GetGlobalDdevDir()
	dest := path.Join(ddevDir, "router-compose.yaml")
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
		return dockerutil.ComposeCmd([]string{dest}, "-p", RouterProjectName, "down", "-v")
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

	// Stop router so we can test port. Of course, it may not be running.
	err = dockerutil.ComposeCmd([]string{dest}, "-p", RouterProjectName, "down")
	if err != nil {
		return fmt.Errorf("failed to stop ddev-router: %v", err)
	}
	err = CheckRouterPorts()
	if err != nil {
		return fmt.Errorf("Unable to listen on required ports, %v", err)
	}

	log.Println("starting ddev router with docker-compose")

	// run docker-compose up -d in the newly created directory
	err = dockerutil.ComposeCmd([]string{dest}, "-p", RouterProjectName, "up", "-d")
	if err != nil {
		return fmt.Errorf("failed to start ddev-router: %v", err)
	}

	fmt.Println("Starting service health checks...")

	// ensure we have a happy router
	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	err = dockerutil.ContainerWait(containerWaitTimeout, label)
	if err != nil {
		return fmt.Errorf("ddev-router failed to become ready: %v", err)
	}

	return nil
}

// PrintRouterStatus outputs router status and warning if not
// running or healthy, as applicable.
// An easy way to make these ports unavailable: sudo netcat -l -p 80  (brew install netcat)
func PrintRouterStatus() string {
	var status string

	badRouter := "\nThe router is not currently running. Your sites are likely inaccessible at this time.\nTry running 'ddev start' on a site to recreate the router."

	label := map[string]string{"com.docker.compose.service": "ddev-router"}
	container, err := dockerutil.FindContainerByLabels(label)

	if err != nil {
		status = color.RedString(SiteNotFound) + badRouter
	} else {
		status = dockerutil.GetContainerHealth(container)
	}

	switch status {
	case "healthy":
		status = color.CyanString(SiteRunning)
	case "exited":
		status = color.RedString(SiteStopped) + badRouter
	default:
		status = color.RedString(status) + badRouter
	}

	return fmt.Sprintf("\nDDEV ROUTER STATUS: %v", status)
}

// determineRouterPorts returns a list of port mappings retrieved from running site
// containers defining VIRTUAL_PORT env var
func determineRouterPorts() []string {
	var routerPorts []string
	containers, err := dockerutil.GetDockerContainers(false)
	if err != nil {
		log.Fatal("failed to retrieve containers for determining port mappings", err)
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

// CheckRouterPorts tries to connect to ports 80/443 as a heuristic to find out
// if they're available for docker to bind to. Returns an error if either one results
// in a successful connection.
func CheckRouterPorts() error {
	// TODO: When we allow configurable ports, we'll want to use an array of configured ports here.
	for _, port := range []int{80, 443} {
		target := fmt.Sprintf("127.0.0.1:%d", port)
		conn, err := net.Dial("tcp", target)
		// We want an error (inability to connect), that's the success case.
		// If we don't get one, return err. This will normally be "getsockopt: connection refused"
		// For simplicity we're not actually studying the err value.
		if err == nil {
			_ = conn.Close()
			return fmt.Errorf("Localhost port %d is in use", port)
		}
	}
	return nil
}
