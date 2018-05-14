package dockerutil

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
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
		output.UserOut.Println("Network", name, "created")

	}
	return nil
}

// EnsureDdevNetwork just creates or ensures the ddev_default network exists or
// exits with fatal.
func EnsureDdevNetwork() {
	// ensure we have docker network
	client := GetDockerClient()
	err := EnsureNetwork(client, NetName)
	if err != nil {
		log.Fatalf("Failed to ensure docker network %s: %v", NetName, err)
	}
}

// GetDockerClient returns a docker client for a docker-machine.
func GetDockerClient() *docker.Client {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		util.Failed("could not get docker client. is docker running? error: %v", err)
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
		return []docker.APIContainers{{}}, err
	}
	containerMatches := []docker.APIContainers{}
	if len(labels) < 1 {
		return []docker.APIContainers{{}}, fmt.Errorf("the provided list of labels was empty")
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
		containerMatches = []docker.APIContainers{{}}
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
// This is modeled on https://gist.github.com/ngauthier/d6e6f80ce977bedca601
func ContainerWait(waittime time.Duration, labels map[string]string) error {

	timeoutChan := time.After(waittime * time.Second)
	tickChan := time.NewTicker(500 * time.Millisecond)
	defer tickChan.Stop()

	status := ""

	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("health check timed out: labels %v timed out without becoming healthy, status=%v", labels, status)

		case <-tickChan.C:
			container, err := FindContainerByLabels(labels)
			if err != nil {
				return fmt.Errorf("failed to query container labels %v", labels)
			}
			status = GetContainerHealth(container)

			if status == "healthy" {
				return nil
			}
		}
	}

	// We should never get here.
	// nolint: vet
	return fmt.Errorf("inappropriate break out of for loop in ContainerWait() waiting for container labels %v", labels)
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
	// "exited" means stopped.
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

// ComposeNoCapture executes a docker-compose command while leaving the stdin/stdout/stderr untouched
// so that people can interact with them directly, for example with ddev ssh. Note that this function
// will never return an actual error because we don't have a way to distinguish between an error
// representing a failure to connect to the container and an error representing a command failing
// inside of the interactive session inside the container.
func ComposeNoCapture(composeFiles []string, action ...string) error {
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

	_ = proc.Run()
	return nil
}

// ComposeCmd executes docker-compose commands via shell.
// returns stdout, stderr, error/nil
func ComposeCmd(composeFiles []string, action ...string) (string, string, error) {
	var arg []string
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	for _, file := range composeFiles {
		arg = append(arg, "-f")
		arg = append(arg, file)
	}

	arg = append(arg, action...)

	proc := exec.Command("docker-compose", arg...)
	proc.Stdout = &stdoutBuf
	proc.Stdin = os.Stdin
	proc.Stderr = &stderrBuf

	var err error
	if err = proc.Start(); err != nil {
		return "", "", fmt.Errorf("Failed to exec docker-compose: %v", err)
	}

	err = proc.Wait()
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("Failed to run docker-compose %v, err='%v', stdoutBuf='%s', stderrBuf='%s'", arg, err, stdoutBuf.String(), stderrBuf.String())
	}

	outStrings := strings.Split(stderrBuf.String(), "\n")
	for _, item := range outStrings {
		line := strings.Trim(item, "\n\r")
		output.UserOut.Println(line)
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
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
		return fmt.Errorf("no docker")
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

// CheckDockerCompose determines if docker-compose is present and executable on the host system. This
// relies on docker-compose being somewhere in the user's $PATH.
func CheckDockerCompose(versionConstraint string) error {
	executableName := "docker-compose"

	path, err := exec.LookPath(executableName)
	if err != nil {
		return fmt.Errorf("no docker-compose")
	}

	out, err := exec.Command(path, "version", "--short").Output()
	if err != nil {
		return err
	}

	version := string(out)
	version = strings.TrimSpace(version)

	dockerComposeVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return err
	}

	match, errs := constraint.Validate(dockerComposeVersion)
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

// CheckForHTTPS determines if a container has the HTTPS_EXPOSE var
// set to route 443 traffic to 80
func CheckForHTTPS(container docker.APIContainers) bool {
	env := GetContainerEnv("HTTPS_EXPOSE", container)
	if env != "" && strings.Contains(env, "443:80") {
		return true
	}
	return false
}
