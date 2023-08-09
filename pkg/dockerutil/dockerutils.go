package dockerutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/archive"
	dockerImages "github.com/ddev/ddev/pkg/docker"
	ddevexec "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
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

// EnsureDdevNetwork just creates or ensures the ddev network exists or
// exits with fatal.
func EnsureDdevNetwork() {
	// ensure we have the fallback global ddev network
	client := GetDockerClient()
	err := EnsureNetwork(client, NetName)
	if err != nil {
		log.Fatalf("Failed to ensure docker network %s: %v", NetName, err)
	}
}

// NetworkExists returns true if the named network exists
// Mostly intended for tests
func NetworkExists(netName string) bool {
	// ensure we have docker network
	client := GetDockerClient()
	return NetExists(client, strings.ToLower(netName))
}

// RemoveNetwork removes the named docker network
func RemoveNetwork(netName string) error {
	client := GetDockerClient()
	err := client.RemoveNetwork(netName)
	return err
}

var DockerHost string
var DockerContext string

// GetDockerClient returns a docker client respecting the current docker context
// but DOCKER_HOST gets priority
func GetDockerClient() *docker.Client {
	var err error

	// This section is skipped if $DOCKER_HOST is set
	if DockerHost == "" {
		DockerContext, DockerHost, err = GetDockerContext()
		// ddev --version may be called without docker client or context available, ignore err
		if err != nil && len(os.Args) > 1 && os.Args[1] != "--version" && os.Args[1] != "hostname" {
			util.Failed("Unable to get docker context: %v", err)
		}
		util.Debug("GetDockerClient: DockerContext=%s, DockerHost=%s", DockerContext, DockerHost)
	}
	// Respect DOCKER_HOST in case it's set, otherwise use host we got from context
	if os.Getenv("DOCKER_HOST") == "" {
		util.Debug("GetDockerClient: Setting DOCKER_HOST to '%s'", DockerHost)
		_ = os.Setenv("DOCKER_HOST", DockerHost)
	}
	client, err := docker.NewClientFromEnv()
	if err != nil {
		output.UserOut.Warnf("could not get docker client. is docker running? error: %v", err)
		// Use os.Exit instead of util.Failed() to avoid import cycle with util.
		os.Exit(100)
	}
	return client
}

// GetDockerContext() returns the currently set docker context, host, and error
func GetDockerContext() (string, string, error) {
	context := ""
	dockerHost := ""

	// This is a cheap way of using docker contexts by running `docker context inspect`
	// I would wish for something far better, but trying to transplant the code from
	// docker/cli did not succeed. rfay 2021-12-16
	// `docker context inspect` will already respect $DOCKER_CONTEXT so we don't have to do that.
	contextInfo, err := ddevexec.RunHostCommand("docker", "context", "inspect", "-f", `{{ .Name }} {{ .Endpoints.docker.Host }}`)
	if err != nil {
		return "", "", fmt.Errorf("unable to run 'docker context inspect' - please make sure docker client is in path and up-to-date: %v", err)
	}
	contextInfo = strings.Trim(contextInfo, " \r\n")
	util.Debug("GetDockerContext: contextInfo='%s'", contextInfo)
	parts := strings.SplitN(contextInfo, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unable to run split docker context info %s: %v", contextInfo, err)
	}
	context = parts[0]
	dockerHost = parts[1]
	util.Debug("Using docker context %s (%v)", context, dockerHost)
	return context, dockerHost, nil
}

// GetDockerHostID returns DOCKER_HOST but with all special characters removed
// It stands in for docker context, but docker context name is not a reliable indicator
func GetDockerHostID() string {
	_, dockerHost, err := GetDockerContext()
	if err != nil {
		util.Warning("Unable to GetDockerContext: %v", err)
	}
	// Make it shorter so we don't hit mutagen 63-char limit
	dockerHost = strings.TrimPrefix(dockerHost, "unix://")
	dockerHost = strings.TrimSuffix(dockerHost, "docker.sock")
	dockerHost = strings.Trim(dockerHost, "/.")
	// Convert remaining descriptor to alphanumeric
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	alphaOnly := reg.ReplaceAllString(dockerHost, "-")
	return alphaOnly
}

// InspectContainer returns the full result of inspection
func InspectContainer(name string) (*docker.Container, error) {
	client, err := docker.NewClientFromEnv()

	if err != nil {
		return nil, err
	}
	c, err := FindContainerByName(name)
	if err != nil || c == nil {
		return nil, err
	}
	x, err := client.InspectContainerWithOptions(docker.InspectContainerOptions{ID: c.ID})
	return x, err
}

// FindContainerByName takes a container name and returns the container
// If container is not found, returns nil with no error
func FindContainerByName(name string) (*docker.APIContainers, error) {
	client := GetDockerClient()

	containers, err := client.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: map[string][]string{"name": {name}},
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, nil
	}

	// ListContainers can return partial matches. Make sure we only match the exact one
	// we're after.
	for _, c := range containers {
		if c.Names[0] == "/"+name {
			return &c, nil
		}
	}
	return nil, nil
}

// GetContainerStateByName returns container state for the named container
func GetContainerStateByName(name string) (string, error) {
	container, err := FindContainerByName(name)
	if err != nil || container == nil {
		return "doesnotexist", fmt.Errorf("container %s does not exist", name)
	}
	if container.State == "running" {
		return container.State, nil
	}
	return container.State, fmt.Errorf("container %s is in state=%s so can't be accessed", name, container.State)
}

// FindContainerByLabels takes a map of label names and values and returns any docker containers which match all labels.
func FindContainerByLabels(labels map[string]string) (*docker.APIContainers, error) {
	containers, err := FindContainersByLabels(labels)
	if err != nil {
		return nil, err
	}
	if len(containers) > 0 {
		return &containers[0], nil
	}
	return nil, nil
}

// GetDockerContainers returns a slice of all docker containers on the host system.
func GetDockerContainers(allContainers bool) ([]docker.APIContainers, error) {
	client := GetDockerClient()
	containers, err := client.ListContainers(docker.ListContainersOptions{All: allContainers})
	return containers, err
}

// FindContainersByLabels takes a map of label names and values and returns any docker containers which match all labels.
// Explanation of the query:
// * docs: https://docs.docker.com/engine/api/v1.23/
// * Stack Overflow: https://stackoverflow.com/questions/28054203/docker-remote-api-filter-exited
func FindContainersByLabels(labels map[string]string) ([]docker.APIContainers, error) {
	if len(labels) < 1 {
		return []docker.APIContainers{{}}, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := []string{}
	for k, v := range labels {
		filterList = append(filterList, fmt.Sprintf("%s=%s", k, v))
	}

	client := GetDockerClient()
	containers, err := client.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: map[string][]string{"label": filterList},
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// FindContainersWithLabel returns all containers with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindContainersWithLabel(label string) ([]docker.APIContainers, error) {
	client := GetDockerClient()
	containers, err := client.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: map[string][]string{"label": {label}},
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
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

// ContainerWait provides a wait loop to check for a single container in "healthy" status.
// waittime is in seconds.
// This is modeled on https://gist.github.com/ngauthier/d6e6f80ce977bedca601
// Returns logoutput, error, returns error if not "healthy"
func ContainerWait(waittime int, labels map[string]string) (string, error) {

	durationWait := time.Duration(waittime) * time.Second
	timeoutChan := time.NewTimer(durationWait)
	tickChan := time.NewTicker(500 * time.Millisecond)
	defer tickChan.Stop()
	defer timeoutChan.Stop()

	status := ""

	for {
		select {
		case <-timeoutChan.C:
			_ = timeoutChan.Stop()
			return "", fmt.Errorf("health check timed out after %v: labels %v timed out without becoming healthy, status=%v", durationWait, labels, status)

		case <-tickChan.C:
			container, err := FindContainerByLabels(labels)
			if err != nil || container == nil {
				return "", fmt.Errorf("failed to query container labels=%v: %v", labels, err)
			}
			health, logOutput := GetContainerHealth(container)

			switch health {
			case "healthy":
				return logOutput, nil
			case "unhealthy":
				return logOutput, fmt.Errorf("container %s unhealthy: %s", container.Names[0], logOutput)
			case "exited":
				service := container.Labels["com.docker.compose.service"]
				suggestedCommand := fmt.Sprintf("ddev logs -s %s", service)
				if service == "ddev-router" || service == "ddev-ssh-agent" {
					suggestedCommand = fmt.Sprintf("docker logs %s", service)
				}
				return logOutput, fmt.Errorf("container exited, please use '%s' to find out why it failed", suggestedCommand)
			}
		}
	}

	// We should never get here.
	//nolint: govet
	return "", fmt.Errorf("inappropriate break out of for loop in ContainerWait() waiting for container labels %v", labels)
}

// ContainersWait provides a wait loop to check for multiple containers in "healthy" status.
// waittime is in seconds.
// Returns logoutput, error, returns error if not "healthy"
func ContainersWait(waittime int, labels map[string]string) error {

	timeoutChan := time.After(time.Duration(waittime) * time.Second)
	tickChan := time.NewTicker(500 * time.Millisecond)
	defer tickChan.Stop()

	status := ""

	for {
		select {
		case <-timeoutChan:
			desc := ""
			containers, err := FindContainersByLabels(labels)
			if err == nil && containers != nil {
				for _, c := range containers {
					health, _ := GetContainerHealth(&c)
					if health != "healthy" {
						n := strings.TrimPrefix(c.Names[0], "/")
						desc = desc + fmt.Sprintf(" %s:%s - more info with `docker inspect --format \"{{json .State.Health }}\" %s`", n, health, n)
					}
				}
			}
			return fmt.Errorf("health check timed out: labels %v timed out without becoming healthy, status=%v, detail=%s ", labels, status, desc)

		case <-tickChan.C:
			containers, err := FindContainersByLabels(labels)
			allHealthy := true
			for _, c := range containers {
				if err != nil || containers == nil {
					return fmt.Errorf("failed to query container labels=%v: %v", labels, err)
				}
				health, logOutput := GetContainerHealth(&c)

				switch health {
				case "healthy":
					continue
				case "unhealthy":
					return fmt.Errorf("container %s is unhealthy: %s", c.Names[0], logOutput)
				case "exited":
					service := c.Labels["com.docker.compose.service"]
					suggestedCommand := fmt.Sprintf("ddev logs -s %s", service)
					if service == "ddev-router" || service == "ddev-ssh-agent" {
						suggestedCommand = fmt.Sprintf("docker logs %s", service)
					}
					return fmt.Errorf("container '%s' exited, please use '%s' to find out why it failed", service, suggestedCommand)
				default:
					allHealthy = false
				}
			}
			if allHealthy {
				return nil
			}
		}
	}

	// We should never get here.
	//nolint: govet
	return fmt.Errorf("inappropriate break out of for loop in ContainerWait() waiting for container labels %v", labels)
}

// ContainerWaitLog provides a wait loop to check for container in "healthy" status.
// with a given log output
// timeout is in seconds.
// This is modeled on https://gist.github.com/ngauthier/d6e6f80ce977bedca601
// Returns logoutput, error, returns error if not "healthy"
func ContainerWaitLog(waittime int, labels map[string]string, expectedLog string) (string, error) {

	timeoutChan := time.After(time.Duration(waittime) * time.Second)
	tickChan := time.NewTicker(500 * time.Millisecond)
	defer tickChan.Stop()

	status := ""

	for {
		select {
		case <-timeoutChan:
			return "", fmt.Errorf("health check timed out: labels %v timed out without becoming healthy, status=%v", labels, status)

		case <-tickChan.C:
			container, err := FindContainerByLabels(labels)
			if err != nil || container == nil {
				return "", fmt.Errorf("failed to query container labels=%v: %v", labels, err)
			}
			status, logOutput := GetContainerHealth(container)

			switch {
			case status == "healthy" && expectedLog == logOutput:
				return logOutput, nil
			case status == "unhealthy":
				return logOutput, fmt.Errorf("container %s unhealthy: %s", container.Names[0], logOutput)
			case status == "exited":
				service := container.Labels["com.docker.compose.service"]
				return logOutput, fmt.Errorf("container exited, please use 'ddev logs -s %s` to find out why it failed", service)
			}
		}
	}

	// We should never get here.
	//nolint: govet
	return "", fmt.Errorf("inappropriate break out of for loop in ContainerWaitLog() waiting for container labels %v", labels)
}

// ContainerName returns the container's human readable name.
func ContainerName(container docker.APIContainers) string {
	return container.Names[0][1:]
}

// GetContainerHealth retrieves the health status of a given container.
// returns status, most-recent-log
func GetContainerHealth(container *docker.APIContainers) (string, string) {
	if container == nil {
		return "no container", ""
	}

	// If the container is not running, then return exited as the health.
	// "exited" means stopped.
	if container.State == "exited" || container.State == "restarting" {
		return container.State, ""
	}

	client := GetDockerClient()
	inspect, err := client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: container.ID,
	})
	if err != nil || inspect == nil {
		output.UserOut.Warnf("Error getting container to inspect: %v", err)
		return "", ""
	}

	logOutput := ""
	status := inspect.State.Health.Status
	// The last log is the most recent
	if inspect.State.Health.Status != "" {
		numLogs := len(inspect.State.Health.Log)
		if numLogs > 0 {
			logOutput = fmt.Sprintf("%v", inspect.State.Health.Log)
		}
	} else {
		// Some containers may not have a healthcheck. In that case
		// we use State to determine health
		switch inspect.State.Status {
		case "running":
			status = "healthy"
		case "exited":
			status = "exited"
		}
	}

	return status, logOutput
}

// ComposeWithStreams executes a docker-compose command but allows the caller to specify
// stdin/stdout/stderr
func ComposeWithStreams(composeFiles []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, action ...string) error {
	defer util.TimeTrack()()

	var arg []string

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return err
	}

	for _, file := range composeFiles {
		arg = append(arg, "-f")
		arg = append(arg, file)
	}

	arg = append(arg, action...)

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return err
	}
	proc := exec.Command(path, arg...)
	proc.Stdout = stdout
	proc.Stdin = stdin
	proc.Stderr = stderr

	err = proc.Run()
	return err
}

// ComposeCmd executes docker-compose commands via shell.
// returns stdout, stderr, error/nil
func ComposeCmd(composeFiles []string, action ...string) (string, string, error) {
	var arg []string
	var stdout bytes.Buffer
	var stderr string

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return "", "", err
	}

	for _, file := range composeFiles {
		arg = append(arg, "-f", file)
	}

	arg = append(arg, action...)

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", "", err
	}
	proc := exec.Command(path, arg...)
	proc.Stdout = &stdout
	proc.Stdin = os.Stdin

	stderrPipe, err := proc.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("Failed to proc.StderrPipe(): %v", err)
	}

	if err = proc.Start(); err != nil {
		return "", "", fmt.Errorf("Failed to exec docker-compose: %v", err)
	}

	// read command's stdout line by line
	in := bufio.NewScanner(stderrPipe)

	// Ignore chatty things from docker-compose like:
	// Container (or Volume) ... Creating or Created or Stopping or Starting or Removing
	// Container Stopped or Created
	// No resource found to remove (when doing a stop and no project exists)
	ignoreRegex := "(^ *(Network|Container|Volume) .* (Creat|Start|Stopp|Remov)ing$|^Container .*(Stopp|Creat)(ed|ing)$|Warning: No resource found to remove$|Pulling fs layer|Waiting|Downloading|Extracting|Verifying Checksum|Download complete|Pull complete)"
	downRE, err := regexp.Compile(ignoreRegex)
	if err != nil {
		util.Warning("failed to compile regex %v: %v", ignoreRegex, err)
	}

	for in.Scan() {
		line := in.Text()
		if len(stderr) > 0 {
			stderr = stderr + "\n"
		}
		stderr = stderr + line
		line = strings.Trim(line, "\n\r")
		switch {
		case downRE.MatchString(line):
			break
		default:
			output.UserOut.Println(line)
		}
	}

	err = proc.Wait()
	if err != nil {
		return stdout.String(), stderr, fmt.Errorf("ComposeCmd failed to run 'COMPOSE_PROJECT_NAME=%s docker-compose %v', action='%v', err='%v', stdout='%s', stderr='%s'", os.Getenv("COMPOSE_PROJECT_NAME"), strings.Join(arg, " "), action, err, stdout.String(), stderr)
	}
	return stdout.String(), stderr, nil
}

// GetAppContainers retrieves docker containers for a given sitename.
func GetAppContainers(sitename string) ([]docker.APIContainers, error) {
	label := map[string]string{"com.ddev.site-name": sitename}
	containers, err := FindContainersByLabels(label)
	if err != nil {
		return containers, err
	}
	return containers, nil
}

// GetContainerEnv returns the value of a given environment variable from a given container.
func GetContainerEnv(key string, container docker.APIContainers) string {
	client := GetDockerClient()
	inspect, err := client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: container.ID,
	})
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
	defer util.TimeTrack()()

	currentVersion, err := GetDockerVersion()
	if err != nil {
		return fmt.Errorf("no docker")
	}
	// If docker version has "_ce", remove it. This happens on OpenSUSE Tumbleweed at least
	currentVersion = strings.TrimSuffix(currentVersion, "_ce")
	dockerVersion, err := semver.NewVersion(currentVersion)
	if err != nil {
		return err
	}

	// See if they're using broken docker desktop on linux
	if runtime.GOOS == "linux" && !nodeps.IsWSL2() {
		client := GetDockerClient()
		info, err := client.Info()
		if err != nil {
			return fmt.Errorf("unable to get docker info: %v", err)
		}
		if info.Name == "docker-desktop" {
			return fmt.Errorf("Docker Desktop on Linux is not yet compatible with DDEV")
		}
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
func CheckDockerCompose() error {
	defer util.TimeTrack()()

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return err
	}
	versionConstraint := DockerComposeVersionConstraint

	v, err := GetDockerComposeVersion()
	if err != nil {
		return err
	}
	dockerComposeVersion, err := semver.NewVersion(v)
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
func GetPublishedPort(privatePort int64, container docker.APIContainers) int {
	for _, port := range container.Ports {
		if port.PrivatePort == privatePort {
			return int(port.PublicPort)
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

var dockerHostRawURL string
var DockerIP string

// GetDockerIP returns either the default Docker IP address (127.0.0.1)
// or the value as configured by $DOCKER_HOST (if DOCKER_HOST is an tcp:// URL)
func GetDockerIP() (string, error) {
	if DockerIP == "" {
		DockerIP = "127.0.0.1"
		dockerHostRawURL = os.Getenv("DOCKER_HOST")
		// If DOCKER_HOST is empty, then the client hasn't been initialized
		// from the docker context
		if dockerHostRawURL == "" {
			_ = GetDockerClient()
			dockerHostRawURL = os.Getenv("DOCKER_HOST")
		}
		if dockerHostRawURL != "" {
			dockerHostURL, err := url.Parse(dockerHostRawURL)
			if err != nil {
				return "", fmt.Errorf("failed to parse $DOCKER_HOST=%s: %v", dockerHostRawURL, err)
			}
			hostPart := dockerHostURL.Hostname()
			if hostPart != "" {
				// Check to see if the hostname we found is an IP address
				addr := net.ParseIP(hostPart)
				if addr == nil {
					// If it wasn't an IP address, look it up to get IP address
					ip, err := net.LookupHost(hostPart)
					if err == nil && len(ip) > 0 {
						hostPart = ip[0]
					} else {
						return "", fmt.Errorf("failed to look up IP address for $DOCKER_HOST=%s, hostname=%s: %v", dockerHostRawURL, hostPart, err)
					}
				}
				DockerIP = hostPart
			}
		}
	}
	return DockerIP, nil
}

// RunSimpleContainer runs a container (non-daemonized) and captures the stdout/stderr.
// It will block, so not to be run on a container whose entrypoint or cmd might hang or run too long.
// This should be the equivalent of something like
// docker run -t -u '%s:%s' -e SNAPSHOT_NAME='%s' -v '%s:/mnt/ddev_config' -v '%s:/var/lib/mysql' --rm --entrypoint=/migrate_file_to_volume.sh %s:%s"
// Example code from https://gist.github.com/fsouza/b0bf3043827f8e39c4589e88cec067d8
// Returns containerID, output, error
func RunSimpleContainer(image string, name string, cmd []string, entrypoint []string, env []string, binds []string, uid string, removeContainerAfterRun bool, detach bool, labels map[string]string, portBindings map[docker.Port][]docker.PortBinding) (containerID string, output string, returnErr error) {
	client := GetDockerClient()

	// Ensure image string includes a tag
	imageChunks := strings.Split(image, ":")
	if len(imageChunks) == 1 {
		// Image does not specify tag
		return "", "", fmt.Errorf("image name must specify tag: %s", image)
	}

	if tag := imageChunks[len(imageChunks)-1]; len(tag) == 0 {
		// Image specifies malformed tag (ends with ':')
		return "", "", fmt.Errorf("malformed tag provided: %s", image)
	}

	existsLocally, err := ImageExistsLocally(image)
	if err != nil {
		return "", "", fmt.Errorf("failed to check if image %s is available locally: %v", image, err)
	}

	if !existsLocally {
		pullErr := Pull(image)
		if pullErr != nil {
			return "", "", fmt.Errorf("failed to pull image %s: %v", image, pullErr)
		}
	}

	// Windows 10 Docker toolbox won't handle a bind mount like C:\..., so must convert to /c/...
	if runtime.GOOS == "windows" {
		for i := range binds {
			binds[i] = strings.Replace(binds[i], `\`, `/`, -1)
			if strings.Index(binds[i], ":") == 1 {
				binds[i] = strings.Replace(binds[i], ":", "", 1)
				binds[i] = "/" + binds[i]
				// And amazingly, the drive letter must be lower-case.
				re := regexp.MustCompile("^/[A-Z]/")
				driveLetter := re.FindString(binds[i])
				if len(driveLetter) == 3 {
					binds[i] = strings.TrimPrefix(binds[i], driveLetter)
					binds[i] = strings.ToLower(driveLetter) + binds[i]
				}

			}
		}
	}

	options := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:        image,
			Cmd:          cmd,
			Env:          env,
			User:         uid,
			Labels:       labels,
			Entrypoint:   entrypoint,
			AttachStderr: true,
			AttachStdout: true,
		},
		HostConfig: &docker.HostConfig{
			Binds:        binds,
			PortBindings: portBindings,
		},
	}

	if runtime.GOOS == "linux" && !IsDockerDesktop() {
		options.HostConfig.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}
	container, err := client.CreateContainer(options)
	if err != nil {
		return "", "", fmt.Errorf("failed to create/start docker container (%v):%v", options, err)
	}

	if removeContainerAfterRun {
		// nolint: errcheck
		defer RemoveContainer(container.ID)
	}
	err = client.StartContainer(container.ID, nil)
	if err != nil {
		return container.ID, "", fmt.Errorf("failed to StartContainer: %v", err)
	}
	exitCode := 0
	if !detach {
		exitCode, err = client.WaitContainer(container.ID)
		if err != nil {
			return container.ID, "", fmt.Errorf("failed to WaitContainer: %v", err)
		}
	}

	// Get logs so we can report them if exitCode failed
	var stdout bytes.Buffer
	err = client.Logs(docker.LogsOptions{
		Stdout:       true,
		Stderr:       true,
		Container:    container.ID,
		OutputStream: &stdout,
		ErrorStream:  &stdout,
	})
	if err != nil {
		return container.ID, "", fmt.Errorf("failed to get Logs(): %v", err)
	}

	// This is the exitCode from the client.WaitContainer()
	if exitCode != 0 {
		return container.ID, stdout.String(), fmt.Errorf("container run failed with exit code %d", exitCode)
	}

	return container.ID, stdout.String(), nil
}

// RemoveContainer stops and removes a container
func RemoveContainer(id string) error {
	client := GetDockerClient()

	err := client.RemoveContainer(docker.RemoveContainerOptions{ID: id, Force: true})
	return err
}

// RestartContainer stops and removes a container
func RestartContainer(id string, timeout uint) error {
	client := GetDockerClient()

	err := client.RestartContainer(id, timeout)
	return err
}

// RemoveContainersByLabels removes all containers that match a set of labels
func RemoveContainersByLabels(labels map[string]string) error {
	client := GetDockerClient()
	containers, err := FindContainersByLabels(labels)
	if err != nil {
		return err
	}
	if containers == nil {
		return nil
	}
	for _, c := range containers {
		err = client.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID, Force: true})
		if err != nil {
			return err
		}
	}
	return nil
}

// ImageExistsLocally determines if an image is available locally.
func ImageExistsLocally(imageName string) (bool, error) {
	client := GetDockerClient()

	// If inspect succeeeds, we have an image.
	_, err := client.InspectImage(imageName)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// Pull pulls image if it doesn't exist locally.
func Pull(imageName string) error {
	exists, err := ImageExistsLocally(imageName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	cmd := exec.Command("docker", "pull", imageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

// GetExposedContainerPorts takes a container pointer and returns an array
// of exposed ports (and error)
func GetExposedContainerPorts(containerID string) ([]string, error) {
	client := GetDockerClient()
	inspectInfo, err := client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: containerID,
	})

	if err != nil {
		return nil, err
	}

	portMap := map[string]bool{}
	for _, portMapping := range inspectInfo.NetworkSettings.Ports {
		if portMapping != nil && len(portMapping) > 0 {
			for _, item := range portMapping {
				portMap[item.HostPort] = true
			}
		}
	}
	ports := []string{}
	for k := range portMap {
		ports = append(ports, k)
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i] < ports[j]
	})
	return ports, nil
}

// MassageWindowsHostMountpoint changes C:/path/to/something to //c/path/to/something
// THis is required for docker bind mounts on docker toolbox.
// Sadly, if we have a Windows drive name, it has to be converted from C:/ to //c for Win10Home/Docker toolbox
func MassageWindowsHostMountpoint(mountPoint string) string {
	if string(mountPoint[1]) == ":" {
		pathPortion := strings.Replace(mountPoint[2:], `\`, "/", -1)
		drive := strings.ToLower(string(mountPoint[0]))
		mountPoint = "/" + drive + pathPortion
	}
	return mountPoint
}

// MassageWindowsNFSMount changes C:\Path\to\something to /c/Path/to/something
func MassageWindowsNFSMount(mountPoint string) string {
	if string(mountPoint[1]) == ":" {
		pathPortion := strings.Replace(mountPoint[2:], `\`, "/", -1)
		drive := string(mountPoint[0])
		// Because we use $HOME to get home in exports, and $HOME has /c/Users/xxx
		// change the drive to lower case.
		mountPoint = "/" + strings.ToLower(drive) + pathPortion
	}
	return mountPoint
}

// RemoveVolume removes named volume. Does not throw error if the volume did not exist.
func RemoveVolume(volumeName string) error {
	client := GetDockerClient()
	if _, err := client.InspectVolume(volumeName); err == nil {
		err := client.RemoveVolumeWithOptions(docker.RemoveVolumeOptions{Name: volumeName})
		if err != nil {
			if err.Error() == "volume in use and cannot be removed" {
				containers, err := client.ListContainers(docker.ListContainersOptions{
					All:     true,
					Filters: map[string][]string{"volume": {volumeName}},
				})
				// Get names of containers which are still using the volume.
				var containerNames []string
				if err == nil {
					for _, container := range containers {
						// Skip first character, it's a slash.
						containerNames = append(containerNames, container.Names[0][1:])
					}
					var containerNamesString = strings.Join(containerNames, " ")
					return fmt.Errorf("Docker volume '%s' is in use by one or more containers and cannot be removed. Use 'docker rm -f %s' to remove them", volumeName, containerNamesString)
				}
				return fmt.Errorf("Docker volume '%s' is in use by a container and cannot be removed. Use 'docker rm -f $(docker ps -aq)' to remove all containers", volumeName)
			}
			return err
		}
	}
	return nil
}

// VolumeExists checks to see if the named volume exists.
func VolumeExists(volumeName string) bool {
	client := GetDockerClient()
	_, err := client.InspectVolume(volumeName)
	if err != nil {
		return false
	}
	return true
}

// VolumeLabels returns map of labels found on volume.
func VolumeLabels(volumeName string) (map[string]string, error) {
	client := GetDockerClient()
	v, err := client.InspectVolume(volumeName)
	if err != nil {
		return nil, err
	}
	return v.Labels, nil
}

// CreateVolume creates a docker volume
func CreateVolume(volumeName string, driver string, driverOpts map[string]string, labels map[string]string) (volume *docker.Volume, err error) {
	client := GetDockerClient()
	volume, err = client.CreateVolume(docker.CreateVolumeOptions{Name: volumeName, Labels: labels, Driver: driver, DriverOpts: driverOpts})
	return volume, err
}

// GetHostDockerInternalIP returns either "" (will use the hostname as is)
// (for Docker Desktop on macOS and Windows with WSL2) or a usable IP address
// But there are many cases to handle
// Linux classic installation
// Gitpod (the Linux technique does not work during prebuild)
// WSL2 with Docker-ce installed inside
// WSL2 with PhpStorm or vscode running inside WSL2
// And it matters whether they're running IDE inside. With docker-inside-wsl2, the bridge docker0 is what we want
// It's also possible to run vscode Language Server inside the web container, in which case host.docker.internal
// should actually be 127.0.0.1
// Inside WSL2, the way to access an app like PhpStorm running on the Windows side is described
// in https://learn.microsoft.com/en-us/windows/wsl/networking#accessing-windows-networking-apps-from-linux-host-ip
// and it involves parsing /etc/resolv.conf.
func GetHostDockerInternalIP() (string, error) {
	hostDockerInternal := ""

	switch {
	case nodeps.IsIPAddress(globalconfig.DdevGlobalConfig.XdebugIDELocation):
		// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
		hostDockerInternal = globalconfig.DdevGlobalConfig.XdebugIDELocation
		util.Debug("host.docker.internal=%s derived from globalconfig.DdevGlobalConfig.XdebugIDELocation", hostDockerInternal)

	case globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationContainer:
		// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
		hostDockerInternal = "127.0.0.1"
		util.Debug("host.docker.internal=%s because globalconfig.DdevGlobalConfig.XdebugIDELocation=%s", hostDockerInternal, globalconfig.XdebugIDELocationContainer)

	case IsColima():
		// Lima just specifies this as a named explicit IP address at this time
		// see https://github.com/lima-vm/lima/blob/master/docs/network.md#host-ip-19216852
		hostDockerInternal = "192.168.5.2"
		util.Debug("host.docker.internal=%s because running on Colima", hostDockerInternal)

	// Gitpod has docker 20.10+ so the docker-compose has already gotten the host-gateway
	case nodeps.IsGitpod():
		util.Debug("host.docker.internal='%s' because on Gitpod", hostDockerInternal)
		break
	case nodeps.IsCodespaces():
		util.Debug("host.docker.internal='%s' because on Codespaces", hostDockerInternal)
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		util.Debug("host.docker.internal='%s' because IsWSL2 and IsDockerDesktop", hostDockerInternal)
		break

	case nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2:
		// If IDE is inside WSL2 then the normal linux processing should work
		util.Debug("host.docker.internal='%s' because globalconfig.DdevGlobalConfig.XdebugIDELocation=%s", hostDockerInternal, globalconfig.XdebugIDELocationWSL2)
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():
		// If IDE is on Windows, we have to parse /etc/resolv.conf
		hostDockerInternal = wsl2ResolvConfNameserver()
		util.Debug("host.docker.internal='%s' because IsWSL2 and !IsDockerDesktop; received from resolv.conf", hostDockerInternal)

	// Docker on linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop) defines host.docker.internal itself.
	case runtime.GOOS == "linux":
		// In docker 20.10+, host.docker.internal is already taken care of by extra_hosts in docker-compose
		util.Debug("host.docker.internal='%s' runtime.GOOS==linux and docker 20.10+", hostDockerInternal)
		break

	default:
		util.Debug("host.docker.internal='%s' because no other case was discovered", hostDockerInternal)
		break
	}

	return hostDockerInternal, nil
}

// GetNFSServerAddr gets the addrss that can be used for the NFS server.
// It's almost the same as GetDockerHostInternalIP() but we have
// to get the actual addr in the case of linux; still, linux rarely
// is used with NFS. Returns "host.docker.internal" by default (not empty)
func GetNFSServerAddr() (string, error) {
	nfsAddr := "host.docker.internal"

	switch {
	case IsColima():
		// Lima just specifies this as a named explicit IP address at this time
		// see https://github.com/lima-vm/lima/blob/master/docs/network.md#host-ip-19216852
		nfsAddr = "192.168.5.2"

	// Gitpod has docker 20.10+ so the docker-compose has already gotten the host-gateway
	// However, NFS will never be used on gitpod.
	case nodeps.IsGitpod():
		break
	case nodeps.IsCodespaces():
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():
		// If IDE is on Windows, we have to parse /etc/resolv.conf
		// Else it will be fine, we can fallthrough to the linux version
		nfsAddr = wsl2ResolvConfNameserver()

	// Docker on linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop) defines host.docker.internal itself.
	case runtime.GOOS == "linux":
		// look up info from the bridge network
		// We can't use the docker host because that's for inside the container,
		// and this is for setting up the network interface
		client := GetDockerClient()
		n, err := client.NetworkInfo("bridge")
		if err != nil {
			return "", err
		}
		if len(n.IPAM.Config) > 0 {
			if n.IPAM.Config[0].Gateway != "" {
				nfsAddr = n.IPAM.Config[0].Gateway
			} else {
				util.Warning("Unable to determine docker bridge gateway - no gateway")
			}
		}
	}

	return nfsAddr, nil
}

// wsl2ResolvConfNameserver parses /etc/resolv.conf to get the nameserver,
// which is the only documented way to know how to connect to the host
// to connect to PhpStorm or other IDE listening there. Or for other apps.
func wsl2ResolvConfNameserver() string {
	if nodeps.IsWSL2() {
		isAuto, err := fileutil.FgrepStringInFile("/etc/resolv.conf", "automatically generated by WSL")
		if err != nil || !isAuto {
			util.Warning("unable to determine WSL2 host.docker.internal because /etc/resolv.conf is not available or not auto-generated")
			return ""
		}
		// We just grepped it so no need to check error
		etcResolv, _ := fileutil.ReadFileIntoString("/etc/resolv.conf")
		util.Debug("resolv.conf=%s", etcResolv)

		nameserverRegex := regexp.MustCompile(`nameserver *([0-9\.]*)`)
		//nameserverRegex.ReplaceAllFunc([]byte(etcResolv), []byte(`$1`))
		res := nameserverRegex.FindStringSubmatch(etcResolv)
		if res == nil || len(res) != 2 {
			util.Warning("unable to determine host.docker.internal from /etc/resolv.conf")
			return ""
		}
		return res[1]
	}
	util.Warning("inappropriately using wsl2ResolvConfNameserver() but not on WSL2")
	return ""
}

// RemoveImage removes an image with force
func RemoveImage(tag string) error {
	client := GetDockerClient()
	_, err := client.InspectImage(tag)
	if err == nil {
		err = client.RemoveImageExtended(tag, docker.RemoveImageOptions{Force: true})

		if err == nil {
			util.Debug("Deleted docker image %s", tag)
		} else {
			util.Warning("Unable to delete %s: %v", tag, err)
		}
	}
	return nil
}

// CopyIntoVolume copies a file or directory on the host into a docker volume
// sourcePath is the host-side full path
// volumeName is the volume name to copy to
// targetSubdir is where to copy it to on the volume
// uid is the uid of the resulting files
// exclusion is a path to be excluded
// If destroyExisting the volume is removed and recreated
func CopyIntoVolume(sourcePath string, volumeName string, targetSubdir string, uid string, exclusion string, destroyExisting bool) error {
	if destroyExisting {
		err := RemoveVolume(volumeName)
		if err != nil {
			util.Warning("could not remove docker volume %s: %v", volumeName, err)
		}
	}
	volPath := "/mnt/v"
	targetSubdirFullPath := volPath + "/" + targetSubdir
	_, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	f, err := os.Open(sourcePath)
	if err != nil {
		util.Failed("Failed to open %s: %v", sourcePath, err)
	}

	// nolint errcheck
	defer f.Close()

	containerName := "CopyIntoVolume_" + nodeps.RandomString(12)

	track := util.TimeTrackC("CopyIntoVolume " + sourcePath + " " + volumeName)
	containerID, _, err := RunSimpleContainer(dockerImages.GetWebImage(), containerName, []string{"sh", "-c", "mkdir -p " + targetSubdirFullPath + " && sleep infinity"}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, map[string]string{"com.ddev.site-name": ""}, nil)
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer RemoveContainer(containerID)

	err = CopyIntoContainer(sourcePath, containerName, targetSubdirFullPath, exclusion)

	if err != nil {
		return err
	}

	// chown/chmod the uploaded content
	c := fmt.Sprintf("chown -R %s %s", uid, targetSubdirFullPath)
	stdout, stderr, err := Exec(containerID, c, "0")
	util.Debug("Exec %s stdout=%s, stderr=%s, err=%v", c, stdout, stderr, err)

	if err != nil {
		return err
	}
	track()
	return nil
}

// Exec does a simple docker exec, no frills, just executes the command
// with the specified uid (or defaults to root=0 if empty uid)
// Returns stdout, stderr, error
func Exec(containerID string, command string, uid string) (string, string, error) {
	client := GetDockerClient()

	if uid == "" {
		uid = "0"
	}
	exec, err := client.CreateExec(docker.CreateExecOptions{
		Container:    containerID,
		Cmd:          []string{"sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         uid,
	})
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = client.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: &stdout,
		ErrorStream:  &stderr,
		Detach:       false,
	})
	if err != nil {
		return "", "", err
	}

	info, err := client.InspectExec(exec.ID)
	if err != nil {
		return stdout.String(), stderr.String(), err
	}
	var execErr error
	if info.ExitCode != 0 {
		execErr = fmt.Errorf("command '%s' returned exit code %v", command, info.ExitCode)
	}

	return stdout.String(), stderr.String(), execErr
}

// CheckAvailableSpace outputs a warning if docker space is low
func CheckAvailableSpace() {
	_, out, _ := RunSimpleContainer(dockerImages.GetWebImage(), "check-available-space-"+util.RandString(6), []string{"sh", "-c", `df / | awk '!/Mounted/ {print $4, $5;}'`}, []string{}, []string{}, []string{}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil)
	out = strings.Trim(out, "% \r\n")
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		util.Warning("Unable to determine docker space usage: %s", out)
		return
	}
	spacePercent, _ := strconv.Atoi(parts[1])
	spaceAbsolute, _ := strconv.Atoi(parts[0]) // Note that this is in KB

	if spaceAbsolute < nodeps.MinimumDockerSpaceWarning {
		util.Error("Your docker install has only %d available disk space, less than %d warning level (%d%% used). Please increase disk image size.", spaceAbsolute, nodeps.MinimumDockerSpaceWarning, spacePercent)
	}
}

// DownloadDockerComposeIfNeeded downloads the proper version of docker-compose
// if it's either not yet installed or has the wrong version.
// Returns downloaded bool (true if it did the download) and err
func DownloadDockerComposeIfNeeded() (bool, error) {
	requiredVersion := globalconfig.GetRequiredDockerComposeVersion()
	var err error
	if requiredVersion == "" {
		util.Debug("globalconfig use_docker_compose_from_path is set, so not downloading")
		return false, nil
	}
	curVersion, err := GetLiveDockerComposeVersion()
	if err != nil || curVersion != requiredVersion {
		err = DownloadDockerCompose()
		if err == nil {
			return true, err
		}
	}
	return false, err
}

// DownloadDockerCompose gets the docker-compose binary and puts it into
// ~/.ddev/.bin
func DownloadDockerCompose() error {
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile, _ := globalconfig.GetDockerComposePath()

	composeURL, err := dockerComposeDownloadLink()
	if err != nil {
		return err
	}
	output.UserOut.Printf("Downloading %s ...", composeURL)

	_ = os.Remove(destFile)

	_ = os.MkdirAll(globalBinDir, 0777)
	err = util.DownloadFile(destFile, composeURL, "true" != os.Getenv("DDEV_NONINTERACTIVE"))
	if err != nil {
		return err
	}
	output.UserOut.Printf("Download complete.")

	// Remove the cached DockerComposeVersion
	globalconfig.DockerComposeVersion = ""

	err = os.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}

func dockerComposeDownloadLink() (string, error) {
	v := globalconfig.GetRequiredDockerComposeVersion()
	if len(v) < 3 {
		return "", fmt.Errorf("required docker-compose version is invalid: %v", v)
	}
	baseVersion := v[1:2]

	switch baseVersion {
	case "2":
		return dockerComposeDownloadLinkV2()
	}
	return "", fmt.Errorf("Invalid docker-compose base version %s", v)
}

// dockerComposeDownloadLinkV2 downlods compose v1 downloads like
//   https://github.com/docker/compose/releases/download/v2.2.1/docker-compose-darwin-aarch64
//   https://github.com/docker/compose/releases/download/v2.2.1/docker-compose-darwin-x86_64
//   https://github.com/docker/compose/releases/download/v2.2.1/docker-compose-windows-x86_64.exe

func dockerComposeDownloadLinkV2() (string, error) {
	arch := runtime.GOARCH

	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	default:
		return "", fmt.Errorf("Only arm64 and amd64 architectures are supported for docker-compose v2, not %s", arch)
	}
	flavor := runtime.GOOS + "-" + arch
	ComposeURL := fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/docker-compose-%s", globalconfig.GetRequiredDockerComposeVersion(), flavor)
	if runtime.GOOS == "windows" {
		ComposeURL = ComposeURL + ".exe"
	}
	return ComposeURL, nil
}

// IsDockerDesktop detects if running on Docker Desktop
func IsDockerDesktop() bool {
	client := GetDockerClient()
	info, err := client.Info()
	if err != nil {
		util.Warning("IsDockerDesktop(): Unable to get docker info, err=%v", err)
		return false
	}
	if info.OperatingSystem == "Docker Desktop" {
		return true
	}
	return false
}

// IsColima detects if running on Colima
func IsColima() bool {
	client := GetDockerClient()
	info, err := client.Info()
	if err != nil {
		util.Warning("IsColima(): Unable to get docker info, err=%v", err)
		return false
	}
	if strings.HasPrefix(info.Name, "colima") {
		return true
	}
	return false
}

// CopyIntoContainer copies a path (file or directory) into a specified container and location
func CopyIntoContainer(srcPath string, containerName string, dstPath string, exclusion string) error {
	startTime := time.Now()
	fi, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	// If a file has been passed in, we'll copy it into a temp directory
	if !fi.IsDir() {
		dirName, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dirName)
		err = fileutil.CopyFile(srcPath, filepath.Join(dirName, filepath.Base(srcPath)))
		if err != nil {
			return err
		}
		srcPath = dirName
	}

	client := GetDockerClient()
	cid, err := FindContainerByName(containerName)
	if err != nil {
		return err
	}
	if cid == nil {
		return fmt.Errorf("CopyIntoContainer unable to find a container named %s", containerName)
	}

	uid, _, _ := util.GetContainerUIDGid()
	_, stderr, err := Exec(cid.ID, "mkdir -p "+dstPath, uid)
	if err != nil {
		return fmt.Errorf("unable to mkdir -p %s inside %s: %v (stderr=%s)", dstPath, containerName, err, stderr)
	}

	tarball, err := os.CreateTemp(os.TempDir(), "containercopytmp*.tar.gz")
	if err != nil {
		return err
	}
	err = tarball.Close()
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer os.Remove(tarball.Name())

	// Tar up the source directory into the tarball
	err = archive.Tar(srcPath, tarball.Name(), exclusion)
	if err != nil {
		return err
	}
	t, err := os.Open(tarball.Name())
	if err != nil {
		return err
	}

	// nolint: errcheck
	defer t.Close()

	err = client.UploadToContainer(cid.ID, docker.UploadToContainerOptions{
		InputStream: t,
		Path:        dstPath,
	})
	if err != nil {
		return err
	}

	util.Debug("Copied %s:%s into %s in %v", srcPath, containerName, dstPath, time.Since(startTime))
	return nil
}

// CopyFromContainer copies a path from a specified container and location to a dstPath on host
func CopyFromContainer(containerName string, containerPath string, hostPath string) error {
	startTime := time.Now()
	err := os.MkdirAll(hostPath, 0755)
	if err != nil {
		return err
	}

	client := GetDockerClient()
	cid, err := FindContainerByName(containerName)
	if err != nil {
		return err
	}
	if cid == nil {
		return fmt.Errorf("CopyFromContainer unable to find a container named %s", containerName)
	}

	f, err := os.CreateTemp("", filepath.Base(hostPath)+".tar.gz")
	if err != nil {
		return err
	}
	//nolint: errcheck
	defer f.Close()
	//nolint: errcheck
	defer os.Remove(f.Name())
	// nolint: errcheck

	err = client.DownloadFromContainer(cid.ID, docker.DownloadFromContainerOptions{
		Path:         containerPath,
		OutputStream: f,
	})
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	err = archive.Untar(f.Name(), hostPath, "")
	if err != nil {
		return err
	}
	util.Success("Copied %s:%s to %s in %v", containerName, containerPath, hostPath, time.Since(startTime))

	return nil
}

// DockerVersionConstraint is the current minimum version of docker required for ddev.
// See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
// REMEMBER TO CHANGE docs/ddev-installation.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/ddev/ddev/pull/738.. and regression https://github.com/ddev/ddev/issues/1431
var DockerVersionConstraint = ">= 20.10.0-alpha1"

// DockerVersion is cached version of docker
var DockerVersion = ""

// GetDockerVersion gets the cached or api-sourced version of docker engine
func GetDockerVersion() (string, error) {
	if DockerVersion != "" {
		return DockerVersion, nil
	}
	client := GetDockerClient()
	if client == nil {
		return "", fmt.Errorf("Unable to get docker version: docker client is nil")
	}

	v, err := client.Version()
	if err != nil {
		return "", err
	}
	DockerVersion = v.Get("Version")

	return DockerVersion, nil
}

// DockerComposeVersionConstraint is the versions allowed for ddev
// REMEMBER TO CHANGE docs/ddev-installation.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/ddev/ddev/pull/738.. and regression https://github.com/ddev/ddev/issues/1431
var DockerComposeVersionConstraint = ">= 2.5.1"

// GetDockerComposeVersion runs docker-compose -v to get the current version
func GetDockerComposeVersion() (string, error) {
	if globalconfig.DockerComposeVersion != "" {
		return globalconfig.DockerComposeVersion, nil
	}

	return GetLiveDockerComposeVersion()
}

// GetLiveDockerComposeVersion runs `docker-compose --version` and caches result
func GetLiveDockerComposeVersion() (string, error) {
	if globalconfig.DockerComposeVersion != "" {
		return globalconfig.DockerComposeVersion, nil
	}

	composePath, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", err
	}

	if !fileutil.FileExists(composePath) {
		globalconfig.DockerComposeVersion = ""
		return globalconfig.DockerComposeVersion, fmt.Errorf("docker-compose does not exist at %s", composePath)
	}
	out, err := exec.Command(composePath, "version", "--short").Output()
	if err != nil {
		return "", err
	}
	v := strings.Trim(string(out), "\r\n")

	// docker-compose v1 and v2.3.3 return a version without the prefix "v", so add it.
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	globalconfig.DockerComposeVersion = v
	return globalconfig.DockerComposeVersion, nil
}
