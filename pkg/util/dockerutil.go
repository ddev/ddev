package util

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/drud/drud-go/utils/dockerutil"
	docker "github.com/fsouza/go-dockerclient"
)

// ContainerWait provides a wait loop to check for container in "healthy" status.
func ContainerWait(timeout time.Duration, labels map[string]string) error {
	timedOut := time.After(timeout * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timedOut:
			return errors.New("health check timed out")
		case <-tick:
			container, err := FindContainerByLabels(labels)
			if err != nil {
				return errors.New("failed to query container")
			}
			status := GetContainerHealth(container)
			if status == "healthy" {
				return nil
			}
		}
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
