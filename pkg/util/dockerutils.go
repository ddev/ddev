package util

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/try"
	"github.com/fsouza/go-dockerclient"
)

// EnsureNetwork will ensure the docker network for ddev is created.
func EnsureNetwork(client *docker.Client, name string) error {
	if !NetExists(client, name) {
		netOptions := docker.CreateNetworkOptions{
			Name:     name,
			Driver:   "bridge",
			Internal: false,
		}
		_, err := client.CreateNetwork(netOptions)
		if err != nil {
			return err
		}
		log.Println("Network", name, "created")

	}
	return nil
}

// GetPort determines and returns the public port for a given container.
func GetPort(name string) (int64, error) {
	client, _ := GetDockerClient()
	var publicPort int64

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return publicPort, err
	}

	for _, ctr := range containers {
		if strings.Contains(ctr.Names[0][1:], name) {
			for _, port := range ctr.Ports {
				if port.PublicPort != 0 {
					publicPort = port.PublicPort
					return publicPort, nil
				}
			}
		}
	}
	return publicPort, fmt.Errorf("%s container not ready", name)
}

// GetPodPort provides a wait loop to help in successfully returning the public port for a given container.
func GetPodPort(name string) (int64, error) {
	var publicPort int64

	err := try.Do(func(attempt int) (bool, error) {
		var err error
		publicPort, err = GetPort(name)
		if err != nil {
			time.Sleep(2 * time.Second) // wait a couple seconds
		}
		return attempt < 70, err
	})
	if err != nil {
		return publicPort, err
	}

	return publicPort, nil
}

// GetDockerClient returns a docker client for a docker-machine.
func GetDockerClient() (*docker.Client, error) {
	// Create a new docker client talking to the default docker-machine.
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		log.Fatal(err)
	}
	return client, err
}

// ProcessContainer will process a docker container for an app listing.
// Since apps contain multiple containers, ProcessContainer will be called once per container.
func ProcessContainer(l map[string]map[string]string, plugin string, containerName string, container docker.APIContainers) {
	label := container.Labels
	appName := label["com.ddev.site-name"]
	appType := label["com.ddev.app-type"]
	containerType := label["com.ddev.container-type"]
	appRoot := label["com.ddev.approot"]
	url := label["com.ddev.app-url"]

	_, exists := l[appName]
	if exists == false {
		l[appName] = map[string]string{
			"name":    appName,
			"status":  container.State,
			"url":     url,
			"type":    appType,
			"approot": appRoot,
		}
	}

	var publicPort int64
	for _, port := range container.Ports {
		if port.PublicPort != 0 {
			publicPort = port.PublicPort
		}
	}

	if containerType == "web" {
		l[appName]["WebPublicPort"] = fmt.Sprintf("%d", publicPort)
	}

	if containerType == "db" {
		l[appName]["DbPublicPort"] = fmt.Sprintf("%d", publicPort)
	}

}

// FindContainerByLabels takes a map of label names and values and returns any docker containers which match all labels.
func FindContainerByLabels(labels map[string]string) (docker.APIContainers, error) {
	client, _ := dockerutil.GetDockerClient()
	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	if len(labels) < 1 {
		return docker.APIContainers{}, fmt.Errorf("the provided list of labels was empty")
	}

	// First, ensure a site name is set and matches the current application.
	for i := range containers {
		matched := true
		for matchName, matchValue := range labels {
			// If the label exists check the value to ensure a match
			if labelValue, ok := containers[i].Labels[matchName]; ok {
				if labelValue != matchValue {
					matched = false
					break
				}
			} else {
				// If the label does not exist, we can just fail immediately.
				matched = false
				break
			}
		}

		if matched {
			return containers[i], nil
		}
	}

	return docker.APIContainers{}, fmt.Errorf("could not find containers which matched search criteria: %+v", labels)
}

// NetExists checks to see if the docker network for ddev exists.
func NetExists(client *docker.Client, name string) bool {
	nets, _ := client.ListNetworks()
	for _, n := range nets {
		if n.Name == name {
			return true
		}
	}
	return false
}

// ContainerWait provides a wait loop to check for container in "healthy" status.
// timeout is in seconds.
func ContainerWait(timeout time.Duration, labels map[string]string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	// doneChan is triggered when we find containers or have an error trying
	doneChan := make(chan bool)
	// timeoutChan is triggered if we are still waiting after timeout seconds
	timeoutChan := time.After(timeout * time.Second)

	// Default error is that it timed out
	var containerErr = errors.New("health check timed out")

	go func() {
		for range ticker.C {
			container, err := FindContainerByLabels(labels)
			if err != nil {
				containerErr = errors.New("failed to query container")
				doneChan <- true
			}
			status := GetContainerHealth(container)
			if status == "healthy" {
				containerErr = nil
				doneChan <- true
			}
		}
	}()

outer:
	for {
		select {

		case <-doneChan:
			break outer
		case <-timeoutChan:
			break outer
		}
	}
	ticker.Stop()
	return containerErr
}

// GetContainerHealth retrieves the status of a given container. The status string returned
// by docker contains uptime and the health status in parenths. This function will filter the uptime and
// return only the health status.
func GetContainerHealth(container docker.APIContainers) string {
	status := container.Status
	re := regexp.MustCompile(`\(([^\)]+)\)`)
	match := re.FindString(status)
	match = strings.Trim(match, "()")
	pre := "health: "
	if strings.HasPrefix(match, pre) {
		match = strings.TrimPrefix(match, pre)
	}
	return match
}
