package platform

import (
	"bytes"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"

	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"

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
		return dockerutil.ComposeCmd([]string{dest}, "-p", routerProjectName, "down")
	}
	return nil
}

// StartDockerRouter ensures the router is running.
func StartDockerRouter() {
	exposedPorts := determineRouterPorts()

	dest := RouterComposeYAMLPath()
	routerdir := filepath.Dir(dest)
	err := os.MkdirAll(routerdir, 0755)
	if err != nil {
		log.Fatalf("unable to create directory for ddev router: %s", err)
	}

	var doc bytes.Buffer
	f, ferr := os.Create(dest)
	if ferr != nil {
		log.Fatal(ferr)
	}
	defer util.CheckClose(f)

	templ := template.New("compose template")
	templ, err = templ.Parse(DrudRouterTemplate)
	if err != nil {
		log.Fatal(ferr)
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
		log.Fatalf("Could not start router: %v", err)
	}
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
