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
		return util.ComposeCmd([]string{dest}, "-p", routerProjectName, "down")
	}
	return nil
}

// StartDockerRouter ensures the router is running.
func StartDockerRouter(ports []string) {
	exposedPorts := SetRouterPorts(ports)

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
	err = util.ComposeCmd([]string{dest}, "-p", routerProjectName, "up", "-d")
	if err != nil {
		log.Fatalf("Could not start router: %v", err)
	}
}

// GetCurrentRouterPorts retrieves the ports currently exposed on the router if it is running.
func GetCurrentRouterPorts() []string {
	var exposedPorts []string
	label := map[string]string{"com.docker.compose.project": "ddevrouter"}
	router, err := util.FindContainerByLabels(label)
	if err == nil {
		ports := router.Ports
		for _, port := range ports {
			if port.PublicPort != 0 {
				exposedPorts = append(exposedPorts, fmt.Sprintf("%v:%v", port.PublicPort, port.PrivatePort))
			}
		}
	}
	return exposedPorts
}

// SetRouterPorts determines the router port configuration based on port definitions from
// site containers and any current configuration from a running router.
func SetRouterPorts(ports []string) []string {
	// Ensure all port values format as hostPort:containerPort
	for i, port := range ports {
		if !strings.Contains(port, ":") {
			ports[i] = port + ":" + port
		}
	}

	// Get any existing router ports and add any from containers not present
	exposedPorts := GetCurrentRouterPorts()
	for _, port := range ports {
		var match bool
		for _, exposed := range exposedPorts {
			if port == exposed {
				match = true
			}
		}
		if !match {
			exposedPorts = append(exposedPorts, port)
		}
	}
	return exposedPorts
}
