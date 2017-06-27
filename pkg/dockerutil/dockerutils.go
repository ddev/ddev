package dockerutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/Masterminds/semver"
	"github.com/fsouza/go-dockerclient"
)

// NetName provides the default network name for ddev.
const NetName = "ddev_default"

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

// GetDockerClient returns a docker client for a docker-machine.
func GetDockerClient() *docker.Client {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatalf("could not get docker client. is docker running? error: %v", err)
	}

	return client
}

// FindContainerByLabels takes a map of label names and values and returns any docker containers which match all labels.
func FindContainerByLabels(labels map[string]string) (docker.APIContainers, error) {
	containers, err := FindContainersByLabels(labels)
	return containers[0], err
}

// GetDockerContainers returns a slice of all docker containers on the host system.
func GetDockerContainers(allContainers bool) ([]docker.APIContainers, error) {
	client := GetDockerClient()
	containers, err := client.ListContainers(docker.ListContainersOptions{All: allContainers})
	return containers, err
}

// FindContainersByLabels takes a map of label names and values and returns any docker containers which match all labels.
func FindContainersByLabels(labels map[string]string) ([]docker.APIContainers, error) {
	var returnError error
	containers, err := GetDockerContainers(true)
	if err != nil {
		return []docker.APIContainers{docker.APIContainers{}}, err
	}
	containerMatches := []docker.APIContainers{}
	if len(labels) < 1 {
		return []docker.APIContainers{docker.APIContainers{}}, fmt.Errorf("the provided list of labels was empty")
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
			containerMatches = append(containerMatches, containers[i])
		}
	}

	// If we couldn't find a match return a list with a single (empty) element alongside the error.
	if len(containerMatches) < 1 {
		containerMatches = []docker.APIContainers{docker.APIContainers{}}
		returnError = fmt.Errorf("could not find containers which matched search criteria: %+v", labels)
	}

	return containerMatches, returnError
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
			if status == "restarting" {
				containerErr = fmt.Errorf("container %s: detected container restart; invalid configuration or container. consider using `docker logs %s` to debug", ContainerName(container), container.ID)
			}
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

// ContainerName returns the containers human readable name.
func ContainerName(container docker.APIContainers) string {
	return container.Names[0][1:]
}

// GetContainerHealth retrieves the status of a given container. The status string returned
// by docker contains uptime and the health status in parenths. This function will filter the uptime and
// return only the health status.
func GetContainerHealth(container docker.APIContainers) string {
	// If the container is not running, then return exited as the health.
	if container.State == "exited" || container.State == "restarting" {
		return container.State
	}

	// Otherwise parse the container status.
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

// ComposeCmd executes docker-compose commands via shell.
func ComposeCmd(composeFiles []string, action ...string) error {
	var arg []string

	for _, file := range composeFiles {
		arg = append(arg, "-f")
		arg = append(arg, file)
	}

	arg = append(arg, action...)

	proc := exec.Command("docker-compose", arg...)
	proc.Stdout = os.Stdout
	proc.Stdin = os.Stdin
	proc.Stderr = os.Stderr

	return proc.Run()
}

// GetAppContainers retrieves docker containers for a given sitename.
func GetAppContainers(sitename string) ([]docker.APIContainers, error) {
	label := map[string]string{"com.ddev.site-name": sitename}
	sites, err := FindContainersByLabels(label)
	if err != nil {
		return sites, err
	}
	return sites, nil
}

// GetContainerEnv returns the value of a given environment variable from a given container.
func GetContainerEnv(key string, container docker.APIContainers) string {
	client := GetDockerClient()
	inspect, err := client.InspectContainer(container.ID)
	if err == nil {
		envVars := inspect.Config.Env

		for _, env := range envVars {
			if strings.HasPrefix(env, key) {
				return strings.TrimPrefix(env, key+"=")
			}
		}
	}
	return ""
}

// CheckDockerVersion determines if the docker version of the host system meets the provided version
// constraints. See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
func CheckDockerVersion(versionConstraint string) error {
	client := GetDockerClient()
	version, err := client.Version()
	if err != nil {
		return err
	}

	currentVersion := version.Get("Version")

	dockerVersion, err := semver.NewVersion(currentVersion)
	if err != nil {
		return err
	}

	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return err
	}

	match, errs := constraint.Validate(dockerVersion)
	if !match {
		if len(errs) <= 1 {
			return errs[0]
		}

		msgs := "\n"
		for _, err := range errs {
			msgs = fmt.Sprint(msgs, err, "\n")
		}
		return fmt.Errorf(msgs)
	}
	return nil
}

// GetPublishedPort returns the published port for a given private port.
func GetPublishedPort(privatePort int64, container docker.APIContainers) int64 {
	for _, port := range container.Ports {
		if port.PrivatePort == privatePort {
			return port.PublicPort
		}
	}
	return 0
}

// GetVolumes returns a list of all docker volumes on the host.
func GetVolumes() ([]docker.Volume, error) {
	client := GetDockerClient()
	return client.ListVolumes(docker.ListVolumesOptions{})
}
