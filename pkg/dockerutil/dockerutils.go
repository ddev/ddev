package dockerutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/archive"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	ddevexec "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerImage "github.com/docker/docker/api/types/image"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerVersions "github.com/docker/docker/api/types/versions"
	dockerVolume "github.com/docker/docker/api/types/volume"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// NetName provides the default network name for ddev.
const NetName = "ddev_default"

type DockerVersionMatrix struct {
	APIVersion string
	Version    string
}

// DockerRequirements defines the minimum Docker version required by DDEV.
// We compare using the APIVersion because it's a consistent and reliable value.
// The Version is displayed to users as it's more readable and user-friendly.
// The values correspond to the API version matrix found here:
// https://docs.docker.com/reference/api/engine/#api-version-matrix
// List of supported Docker versions: https://endoflife.date/docker-engine
var DockerRequirements = DockerVersionMatrix{
	APIVersion: "1.44",
	Version:    "25.0",
}

type ComposeCmdOpts struct {
	ComposeFiles []string
	Profiles     []string
	Action       []string
	Progress     bool // Add dots every second while the compose command is running
	Timeout      time.Duration
}

// NoHealthCheck is a HealthConfig that disables any existing healthcheck when
// running a container. Used by RunSimpleContainer
// See https://pkg.go.dev/github.com/moby/docker-image-spec/specs-go/v1#HealthcheckConfig
var NoHealthCheck = dockerContainer.HealthConfig{
	Test: []string{"NONE"}, // Disables any existing health check
}

// EnsureNetwork will ensure the Docker network for DDEV is created.
func EnsureNetwork(ctx context.Context, client *dockerClient.Client, name string, netOptions dockerNetwork.CreateOptions) error {
	// Pre-check for network duplicates
	RemoveNetworkDuplicates(ctx, client, name)

	if !NetExists(ctx, client, name) {
		_, err := client.NetworkCreate(ctx, name, netOptions)
		if err != nil {
			return err
		}
		output.UserOut.Println("Network", name, "created")
	}
	return nil
}

// EnsureDdevNetwork creates or ensures the DDEV network exists or
// exits with fatal.
func EnsureDdevNetwork() {
	// Ensure we have the fallback global DDEV network
	ctx, client := GetDockerClient()
	netOptions := dockerNetwork.CreateOptions{
		Driver:   "bridge",
		Internal: false,
		Labels:   map[string]string{"com.ddev.platform": "ddev"},
	}
	err := EnsureNetwork(ctx, client, NetName, netOptions)
	if err != nil {
		log.Fatalf("Failed to ensure Docker network %s: %v", NetName, err)
	}
}

// NetworkExists returns true if the named network exists
// Mostly intended for tests
func NetworkExists(netName string) bool {
	// Ensure we have Docker network
	ctx, client := GetDockerClient()
	return NetExists(ctx, client, strings.ToLower(netName))
}

// RemoveNetwork removes the named Docker network
// netName can also be network's ID
func RemoveNetwork(netName string) error {
	ctx, client := GetDockerClient()
	networks, _ := client.NetworkList(ctx, dockerNetwork.ListOptions{})
	// the loop below may not contain such a network
	var err = errdefs.NotFound(errors.New("not found"))
	// loop through all networks because there may be duplicates
	// and delete only by ID - it's unique, but the name isn't
	for _, network := range networks {
		if network.Name == netName || network.ID == netName {
			err = client.NetworkRemove(ctx, network.ID)
		}
	}
	return err
}

// RemoveNetworkWithWarningOnError removes the named Docker network
func RemoveNetworkWithWarningOnError(netName string) {
	err := RemoveNetwork(netName)
	// If it's a "no such network" there's no reason to report error
	if err != nil && !IsErrNotFound(err) {
		util.Warning("Unable to remove network %s: %v", netName, err)
	} else if err == nil {
		output.UserOut.Println("Network", netName, "removed")
	}
}

// RemoveNetworkDuplicates removes the duplicates for the named Docker network
// This means that if there is only one network with this name - no action,
// and if there are several such networks, then we leave the first one, and delete the others
func RemoveNetworkDuplicates(ctx context.Context, client *dockerClient.Client, netName string) {
	networks, _ := client.NetworkList(ctx, dockerNetwork.ListOptions{})
	networkMatchFound := false
	for _, network := range networks {
		if network.Name == netName || network.ID == netName {
			if networkMatchFound == true {
				err := client.NetworkRemove(ctx, network.ID)
				// If it's a "no such network" there's no reason to report error
				if err != nil && !IsErrNotFound(err) {
					util.Warning("Unable to remove network %s: %v", netName, err)
				}
			} else {
				networkMatchFound = true
			}
		}
	}
}

var DockerHost string
var DockerContext string
var DockerCtx context.Context
var DockerClient *dockerClient.Client

// GetDockerClient returns a Docker client respecting the current Docker context
// but DOCKER_HOST gets priority
func GetDockerClient() (context.Context, *dockerClient.Client) {
	var err error

	// This section is skipped if $DOCKER_HOST is set
	if DockerHost == "" {
		DockerContext, DockerHost, err = GetDockerContext()
		// ddev --version may be called without Docker client or context available, ignore err
		if err != nil && !CanRunWithoutDocker() {
			util.Failed("Unable to get Docker context: %v", err)
		}
		util.Debug("GetDockerClient: DockerContext=%s, DockerHost=%s", DockerContext, DockerHost)
	}
	// Respect DOCKER_HOST in case it's set, otherwise use host we got from context
	if os.Getenv("DOCKER_HOST") == "" {
		util.Debug("GetDockerClient: Setting DOCKER_HOST to '%s'", DockerHost)
		_ = os.Setenv("DOCKER_HOST", DockerHost)
	}
	if DockerClient == nil {
		DockerCtx = context.Background()
		DockerClient, err = dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
		if err != nil {
			output.UserOut.Warnf("Could not get Docker client. Is Docker running? Error: %v", err)
			// Use os.Exit instead of util.Failed() to avoid import cycle with util.
			os.Exit(100)
		}
		defer DockerClient.Close()
	}
	return DockerCtx, DockerClient
}

// GetDockerContext returns the currently set Docker context, host, and error
func GetDockerContext() (string, string, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return "", "", fmt.Errorf("unable to get Docker CLI client: %v", err)
	}
	opts := flags.NewClientOptions()
	if err := dockerCli.Initialize(opts); err != nil {
		return "", "", fmt.Errorf("unable to initialize Docker CLI client: %v", err)
	}
	dockerContext := dockerCli.CurrentContext()
	dockerHost := dockerCli.DockerEndpoint().Host
	util.Debug("Using Docker context %s (%v)", dockerContext, dockerHost)
	return dockerContext, dockerHost, nil
}

// GetDockerHostID returns DOCKER_HOST but with all special characters removed
// It stands in for Docker context, but Docker context name is not a reliable indicator
func GetDockerHostID() string {
	_, dockerHost, err := GetDockerContext()
	if err != nil {
		util.Warning("Unable to GetDockerContext: %v", err)
	}
	// Make it shorter so we don't hit Mutagen 63-char limit
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
func InspectContainer(name string) (dockerContainer.InspectResponse, error) {
	ctx := context.Background()
	client, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())

	if err != nil {
		return dockerContainer.InspectResponse{}, err
	}
	container, err := FindContainerByName(name)
	if err != nil || container == nil {
		return dockerContainer.InspectResponse{}, err
	}
	x, err := client.ContainerInspect(ctx, container.ID)
	return x, err
}

// FindContainerByName takes a container name and returns the container
// If container is not found, returns nil with no error
func FindContainerByName(name string) (*dockerContainer.Summary, error) {
	ctx, client := GetDockerClient()

	containers, err := client.ContainerList(ctx, dockerContainer.ListOptions{
		All:     true,
		Filters: dockerFilters.NewArgs(dockerFilters.KeyValuePair{Key: "name", Value: name}),
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, nil
	}

	// ListContainers can return partial matches. Make sure we only match the exact one
	// we're after.
	for _, container := range containers {
		if container.Names[0] == "/"+name {
			return &container, nil
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

// FindContainerByLabels takes a map of label names and values and returns any Docker containers which match all labels.
func FindContainerByLabels(labels map[string]string) (*dockerContainer.Summary, error) {
	containers, err := FindContainersByLabels(labels)
	if err != nil {
		return nil, err
	}
	if len(containers) > 0 {
		return &containers[0], nil
	}
	return nil, nil
}

// GetDockerContainers returns a slice of all Docker containers on the host system.
func GetDockerContainers(allContainers bool) ([]dockerContainer.Summary, error) {
	ctx, client := GetDockerClient()
	containers, err := client.ContainerList(ctx, dockerContainer.ListOptions{All: allContainers})
	return containers, err
}

// FindContainersByLabels takes a map of label names and values and returns any Docker containers which match all labels.
// Explanation of the query:
// * docs: https://docs.docker.com/engine/api/v1.23/
// * Stack Overflow: https://stackoverflow.com/questions/28054203/docker-remote-api-filter-exited
func FindContainersByLabels(labels map[string]string) ([]dockerContainer.Summary, error) {
	if len(labels) < 1 {
		return []dockerContainer.Summary{{}}, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := dockerFilters.NewArgs()
	for k, v := range labels {
		label := fmt.Sprintf("%s=%s", k, v)
		// If no value is specified, filter any value by the key.
		if v == "" {
			label = k
		}
		filterList.Add("label", label)
	}

	ctx, client := GetDockerClient()
	containers, err := client.ContainerList(ctx, dockerContainer.ListOptions{
		All:     true,
		Filters: filterList,
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// FindContainersWithLabel returns all containers with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindContainersWithLabel(label string) ([]dockerContainer.Summary, error) {
	ctx, client := GetDockerClient()
	containers, err := client.ContainerList(ctx, dockerContainer.ListOptions{
		All:     true,
		Filters: dockerFilters.NewArgs(dockerFilters.KeyValuePair{Key: "label", Value: label}),
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
}

// FindImagesByLabels takes a map of label names and values and returns any Docker images which match all labels.
// danglingOnly is used to return only dangling images, otherwise return all of them, including dangling.
func FindImagesByLabels(labels map[string]string, danglingOnly bool) ([]dockerImage.Summary, error) {
	if len(labels) < 1 {
		return []dockerImage.Summary{{}}, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := dockerFilters.NewArgs()
	for k, v := range labels {
		label := fmt.Sprintf("%s=%s", k, v)
		// If no value is specified, filter any value by the key.
		if v == "" {
			label = k
		}
		filterList.Add("label", label)
	}

	if danglingOnly {
		filterList.Add("dangling", "true")
	}

	ctx, client := GetDockerClient()
	images, err := client.ImageList(ctx, dockerImage.ListOptions{
		All:     true,
		Filters: filterList,
	})
	if err != nil {
		return nil, err
	}
	return images, nil
}

// NetExists checks to see if the Docker network for DDEV exists.
func NetExists(ctx context.Context, client *dockerClient.Client, name string) bool {
	nets, _ := client.NetworkList(ctx, dockerNetwork.ListOptions{})
	for _, n := range nets {
		if n.Name == name {
			return true
		}
	}
	return false
}

// FindNetworksWithLabel returns all networks with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindNetworksWithLabel(label string) ([]dockerNetwork.Inspect, error) {
	ctx, client := GetDockerClient()
	networks, err := client.NetworkList(ctx, dockerNetwork.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matchingNetworks []dockerNetwork.Inspect
	for _, network := range networks {
		if network.Labels != nil {
			if _, exists := network.Labels[label]; exists {
				matchingNetworks = append(matchingNetworks, network)
			}
		}
	}

	return matchingNetworks, nil
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
			desc := ""
			container, err := FindContainerByLabels(labels)
			if err == nil && container != nil {
				health, _ := GetContainerHealth(container)
				if health != "healthy" {
					name, suggestedCommand := getSuggestedCommandForContainerLog(container)
					desc = desc + fmt.Sprintf(" %s:%s\nTroubleshoot this with these commands:\n%s", name, health, suggestedCommand)
				}
			}
			return "", fmt.Errorf("health check timed out after %v: labels %v timed out without becoming healthy, status=%v, detail=%s ", durationWait, labels, status, desc)

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
				name, suggestedCommand := getSuggestedCommandForContainerLog(container)
				return logOutput, fmt.Errorf("%s container is unhealthy, log=%s\nTroubleshoot this with these commands: \n%s", name, logOutput, suggestedCommand)
			case "exited":
				name, suggestedCommand := getSuggestedCommandForContainerLog(container)
				return logOutput, fmt.Errorf("%s container exited,\nTroubleshoot this with these commands:\n%s", name, suggestedCommand)
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
				for _, container := range containers {
					health, _ := GetContainerHealth(&container)
					if health != "healthy" {
						name, suggestedCommand := getSuggestedCommandForContainerLog(&container)
						desc = desc + fmt.Sprintf(" %s:%s\nTroubleshoot this with these commands:\n%s", name, health, suggestedCommand)
					}
				}
			}
			return fmt.Errorf("health check timed out: labels %v timed out without becoming healthy, status=%v, detail=%s ", labels, status, desc)

		case <-tickChan.C:
			containers, err := FindContainersByLabels(labels)
			if err != nil || containers == nil {
				return fmt.Errorf("failed to query container labels=%v: %v", labels, err)
			}
			allHealthy := true
			for _, container := range containers {
				health, logOutput := GetContainerHealth(&container)

				switch health {
				case "healthy":
					continue
				case "unhealthy":
					name, suggestedCommand := getSuggestedCommandForContainerLog(&container)
					return fmt.Errorf("%s container is unhealthy, log=%s\nTroubleshoot this with these commands:\n%s", name, logOutput, suggestedCommand)
				case "exited":
					name, suggestedCommand := getSuggestedCommandForContainerLog(&container)
					return fmt.Errorf("%s container exited.\nTroubleshoot this with these commands:\n%s", name, suggestedCommand)
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
			desc := ""
			container, err := FindContainerByLabels(labels)
			if err == nil && container != nil {
				health, _ := GetContainerHealth(container)
				if health != "healthy" {
					name, suggestedCommand := getSuggestedCommandForContainerLog(container)
					desc = desc + fmt.Sprintf(" %s:%s\nTroubleshoot this with these commands:\n%s", name, health, suggestedCommand)
				}
			}
			return "", fmt.Errorf("health check timed out: labels %v timed out without becoming healthy, status=%v, detail=%s ", labels, status, desc)

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
				name, suggestedCommand := getSuggestedCommandForContainerLog(container)
				return logOutput, fmt.Errorf("%s container is unhealthy, log=%s\nTroubleshoot this with these commands:\n%s", name, logOutput, suggestedCommand)
			case status == "exited":
				name, suggestedCommand := getSuggestedCommandForContainerLog(container)
				return logOutput, fmt.Errorf("%s container exited\nTroubleshoot this with these commands:\n%s", name, suggestedCommand)
			}
		}
	}

	// We should never get here.
	//nolint: govet
	return "", fmt.Errorf("inappropriate break out of for loop in ContainerWaitLog() waiting for container labels %v", labels)
}

// getSuggestedCommandForContainerLog returns a command that can be used to find out what is wrong with a container
func getSuggestedCommandForContainerLog(container *dockerContainer.Summary) (string, string) {
	suggestedCommands := []string{}
	service := container.Labels["com.docker.compose.service"]
	if service != "" && service != "ddev-router" && service != "ddev-ssh-agent" {
		suggestedCommands = append(suggestedCommands, fmt.Sprintf("ddev logs -s %s", service))
	}
	name := strings.TrimPrefix(container.Names[0], "/")
	if name != "" {
		suggestedCommands = append(suggestedCommands, fmt.Sprintf("docker logs %s", name), fmt.Sprintf("docker inspect --format \"{{ json .State.Health }}\" %s | docker run -i --rm ddev/ddev-utilities jq -r", name))
	}
	// Should never happen, but added just in case
	if name == "" {
		name = "unknown"
	}
	if len(suggestedCommands) == 0 {
		suggestedCommands = append(suggestedCommands, "ddev logs", "docker logs CONTAINER (find CONTAINER with 'docker ps')", "docker inspect --format \"{{ json .State.Health }}\" CONTAINER", "docker inspect --format \"{{ json .State.Health }}\" CONTAINER | docker run -i --rm ddev/ddev-utilities jq -r")
	}
	suggestedCommand, _ := util.ArrayToReadableOutput(suggestedCommands)
	return name, suggestedCommand
}

// ContainerName returns the container's human-readable name.
func ContainerName(container dockerContainer.Summary) string {
	return container.Names[0][1:]
}

// GetContainerHealth retrieves the health status of a given container.
// returns status, most-recent-log
func GetContainerHealth(container *dockerContainer.Summary) (string, string) {
	if container == nil {
		return "no container", ""
	}

	// If the container is not running, then return exited as the health.
	// "exited" means stopped.
	if container.State == "exited" || container.State == "restarting" {
		return container.State, ""
	}

	ctx, client := GetDockerClient()
	inspect, err := client.ContainerInspect(ctx, container.ID)
	if err != nil {
		output.UserOut.Warnf("Error getting container to inspect: %v", err)
		return "", ""
	}

	logOutput := ""
	status := ""
	if inspect.State.Health != nil {
		status = inspect.State.Health.Status
	}
	// The last log is the most recent
	if status != "" {
		numLogs := len(inspect.State.Health.Log)
		if numLogs > 0 {
			logOutput = fmt.Sprintf("%v", inspect.State.Health.Log[numLogs-1].Output)
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

	return status, strings.TrimSpace(logOutput)
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
func ComposeCmd(cmd *ComposeCmdOpts) (string, string, error) {
	var arg []string
	var stdout bytes.Buffer
	var stderr string

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return "", "", err
	}

	for _, file := range cmd.ComposeFiles {
		arg = append(arg, "-f", file)
	}

	for _, profile := range cmd.Profiles {
		arg = append(arg, "--profile", profile)
	}

	arg = append(arg, cmd.Action...)

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}
	proc := exec.CommandContext(ctx, path, arg...)
	proc.Stdout = &stdout
	proc.Stdin = os.Stdin

	stderrPipe, err := proc.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to proc.StderrPipe(): %v", err)
	}

	if err = proc.Start(); err != nil {
		return "", "", fmt.Errorf("failed to exec docker-compose: %v", err)
	}

	stderrOutput := bufio.NewScanner(stderrPipe)

	// Ignore chatty things from docker-compose like:
	// Container (or Volume) ... Creating or Created or Stopping or Starting or Removing
	// Container Stopped or Created
	// No resource found to remove (when doing a stop and no project exists)
	ignoreRegex := "(^ *(Network|Container|Volume|Service) .* (Creat|Start|Stopp|Remov|Build|Buil|Runn)(ing|t)$|.* Built$|^Container .*(Stopp|Creat)(ed|ing)$|Warning: No resource found to remove|Pulling fs layer|Waiting|Downloading|Extracting|Verifying Checksum|Download complete|Pull complete)"
	downRE, err := regexp.Compile(ignoreRegex)
	if err != nil {
		util.Warning("Failed to compile regex %v: %v", ignoreRegex, err)
	}

	var done chan bool
	if cmd.Progress {
		done = util.ShowDots()
	}
	for stderrOutput.Scan() {
		line := stderrOutput.Text()
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
	if cmd.Progress {
		done <- true
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return stdout.String(), stderr, fmt.Errorf("composeCmd timed out after %v and failed to run 'COMPOSE_PROJECT_NAME=%s docker-compose %v', action='%v', err='%v', stdout='%s', stderr='%s'", cmd.Timeout, os.Getenv("COMPOSE_PROJECT_NAME"), strings.Join(arg, " "), cmd.Action, err, stdout.String(), stderr)
	}
	if err != nil {
		return stdout.String(), stderr, fmt.Errorf("composeCmd failed to run 'COMPOSE_PROJECT_NAME=%s docker-compose %v', action='%v', err='%v', stdout='%s', stderr='%s'", os.Getenv("COMPOSE_PROJECT_NAME"), strings.Join(arg, " "), cmd.Action, err, stdout.String(), stderr)
	}
	return stdout.String(), stderr, nil
}

// GetAppContainers retrieves docker containers for a given sitename.
func GetAppContainers(sitename string) ([]dockerContainer.Summary, error) {
	label := map[string]string{
		"com.ddev.site-name":        sitename,
		"com.docker.compose.oneoff": "False",
	}
	containers, err := FindContainersByLabels(label)
	if err != nil {
		return containers, err
	}
	return containers, nil
}

// GetContainerEnv returns the value of a given environment variable from a given container.
func GetContainerEnv(key string, container dockerContainer.Summary) string {
	ctx, client := GetDockerClient()
	inspect, err := client.ContainerInspect(ctx, container.ID)

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

// CheckDockerVersion determines if the Docker version of the host system meets the provided
// minimum for the Docker API Version.
func CheckDockerVersion(dockerVersionMatrix DockerVersionMatrix) error {
	defer util.TimeTrack()()

	currentVersion, err := GetDockerVersion()
	if err != nil {
		return fmt.Errorf("no docker")
	}
	currentAPIVersion, err := GetDockerAPIVersion()
	if err != nil {
		return fmt.Errorf("no docker")
	}

	// See if they're using broken Docker Desktop on Linux
	if runtime.GOOS == "linux" && !nodeps.IsWSL2() {
		ctx, client := GetDockerClient()
		info, err := client.Info(ctx)
		if err != nil {
			return fmt.Errorf("unable to get Docker info: %v", err)
		}
		if info.Name == "docker-desktop" {
			return fmt.Errorf("docker desktop on Linux is not yet compatible with DDEV")
		}
	}

	// Check against recommended API version, if it fails, suggest the minimum Docker version that relates to supported API
	if !dockerVersions.GreaterThanOrEqualTo(currentAPIVersion, dockerVersionMatrix.APIVersion) {
		return fmt.Errorf("installed Docker version %s is not supported, please update to version %s or newer", currentVersion, dockerVersionMatrix.Version)
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
		return fmt.Errorf("%s", msgs)
	}

	return nil
}

// GetPublishedPort returns the published port for a given private port.
func GetPublishedPort(privatePort uint16, container dockerContainer.Summary) int {
	for _, port := range container.Ports {
		if port.PrivatePort == privatePort {
			return int(port.PublicPort)
		}
	}
	return 0
}

// CheckForHTTPS determines if a container has the HTTPS_EXPOSE var
// set to route 443 traffic to 80
func CheckForHTTPS(container dockerContainer.Summary) bool {
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
		// from the Docker context
		if dockerHostRawURL == "" {
			_, _ = GetDockerClient()
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
					ip, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", hostPart)
					if err == nil && len(ip) > 0 {
						hostPart = ip[0].String()
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
// docker run -t -u '%s:%s' -e SNAPSHOT_NAME='%s' -v '%s:/mnt/ddev_config' -v '%s:/var/lib/mysql' --no-healthcheck --rm --entrypoint=/migrate_file_to_volume.sh %s:%s"
// Example code from https://gist.github.com/fsouza/b0bf3043827f8e39c4589e88cec067d8
// Default behavior is to use the image's healthcheck (healthConfig == nil)
// When passed a pointer to HealthConfig (often &dockerutils.NoHealthCheck) it can turn off healthcheck
// or it can replace it or have other behaviors, see
// https://pkg.go.dev/github.com/moby/docker-image-spec/specs-go/v1#HealthcheckConfig
// Returns containerID, output, error
func RunSimpleContainer(image string, name string, cmd []string, entrypoint []string, env []string, binds []string, uid string, removeContainerAfterRun bool, detach bool, labels map[string]string, portBindings nat.PortMap, healthConfig *dockerContainer.HealthConfig) (containerID string, output string, returnErr error) {
	ctx, client := GetDockerClient()

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

	containerConfig := &dockerContainer.Config{
		Image:        image,
		Cmd:          cmd,
		Env:          env,
		User:         uid,
		Labels:       labels,
		Entrypoint:   entrypoint,
		AttachStderr: true,
		AttachStdout: true,
		Healthcheck:  healthConfig,
	}

	containerHostConfig := &dockerContainer.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
	}

	if runtime.GOOS == "linux" && !IsDockerDesktop() {
		containerHostConfig.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}

	container, err := client.ContainerCreate(ctx, containerConfig, containerHostConfig, nil, nil, name)
	if err != nil {
		return "", "", fmt.Errorf("failed to create/start Docker container %v (%v, %v): %v", name, containerConfig, containerHostConfig, err)
	}

	if removeContainerAfterRun {
		// nolint: errcheck
		defer RemoveContainer(container.ID)
	}

	err = client.ContainerStart(ctx, container.ID, dockerContainer.StartOptions{})
	if err != nil {
		return container.ID, "", fmt.Errorf("failed to StartContainer: %v", err)
	}

	exitCode := 0
	if !detach {
		waitChan, errChan := client.ContainerWait(ctx, container.ID, "")
		select {
		case status := <-waitChan:
			exitCode = int(status.StatusCode)
		case err := <-errChan:
			return container.ID, "", fmt.Errorf("failed to ContainerWait: %v", err)
		}
	}

	// Get logs so we can report them if exitCode failed
	var stdout bytes.Buffer
	options := dockerContainer.LogsOptions{ShowStdout: true, ShowStderr: true}
	rc, err := client.ContainerLogs(ctx, container.ID, options)
	if err != nil {
		return container.ID, "", fmt.Errorf("failed to get container logs: %v", err)
	}
	defer rc.Close()

	_, err = stdcopy.StdCopy(&stdout, &stdout, rc)
	if err != nil {
		util.Warning("failed to copy container logs: %v", err)
	}

	// This is the exitCode from the cli.ContainerWait()
	if exitCode != 0 {
		return container.ID, stdout.String(), fmt.Errorf("container run failed with exit code %d", exitCode)
	}

	return container.ID, stdout.String(), nil
}

// RemoveContainer stops and removes a container
func RemoveContainer(id string) error {
	ctx, client := GetDockerClient()

	err := client.ContainerRemove(ctx, id, dockerContainer.RemoveOptions{Force: true})
	return err
}

// RestartContainer stops and removes a container
func RestartContainer(id string, timeout *int) error {
	ctx, client := GetDockerClient()

	err := client.ContainerRestart(ctx, id, dockerContainer.StopOptions{Timeout: timeout})
	return err
}

// RemoveContainersByLabels removes all containers that match a set of labels
func RemoveContainersByLabels(labels map[string]string) error {
	ctx, client := GetDockerClient()
	containers, err := FindContainersByLabels(labels)
	if err != nil {
		return err
	}
	if containers == nil {
		return nil
	}
	for _, container := range containers {
		err = client.ContainerRemove(ctx, container.ID, dockerContainer.RemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}
	return nil
}

// ImageExistsLocally determines if an image is available locally.
func ImageExistsLocally(imageName string) (bool, error) {
	ctx, client := GetDockerClient()

	// If inspect succeeds, we have an image.
	_, err := client.ImageInspect(ctx, imageName)
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

// GetBoundHostPorts takes a container pointer and returns an array
// of exposed ports (and error)
func GetBoundHostPorts(containerID string) ([]string, error) {
	ctx, client := GetDockerClient()
	inspectInfo, err := client.ContainerInspect(ctx, containerID)

	if err != nil {
		return nil, err
	}

	portMap := map[string]bool{}

	if inspectInfo.HostConfig != nil && inspectInfo.HostConfig.PortBindings != nil {
		for _, portBindings := range inspectInfo.HostConfig.PortBindings {
			if len(portBindings) > 0 {
				for _, binding := range portBindings {
					// Only include ports with a non-empty HostPort
					if binding.HostPort != "" {
						portMap[binding.HostPort] = true
					}
				}
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
	ctx, client := GetDockerClient()
	if _, err := client.VolumeInspect(ctx, volumeName); err == nil {
		err := client.VolumeRemove(ctx, volumeName, true)
		if err != nil {
			if err.Error() == "volume in use and cannot be removed" {
				containers, err := client.ContainerList(ctx, dockerContainer.ListOptions{
					All:     true,
					Filters: dockerFilters.NewArgs(dockerFilters.KeyValuePair{Key: "volume", Value: volumeName}),
				})
				// Get names of containers which are still using the volume.
				var containerNames []string
				if err == nil {
					for _, container := range containers {
						// Skip first character, it's a slash.
						containerNames = append(containerNames, container.Names[0][1:])
					}
					var containerNamesString = strings.Join(containerNames, " ")
					return fmt.Errorf("docker volume '%s' is in use by one or more containers and cannot be removed. Use 'docker rm -f %s' to remove them", volumeName, containerNamesString)
				}
				return fmt.Errorf("docker volume '%s' is in use by a container and cannot be removed. Use 'docker rm -f $(docker ps -aq)' to remove all containers", volumeName)
			}
			return err
		}
	}
	return nil
}

// VolumeExists checks to see if the named volume exists.
func VolumeExists(volumeName string) bool {
	ctx, client := GetDockerClient()
	_, err := client.VolumeInspect(ctx, volumeName)
	if err != nil {
		return false
	}
	return true
}

// VolumeLabels returns map of labels found on volume.
func VolumeLabels(volumeName string) (map[string]string, error) {
	ctx, client := GetDockerClient()
	v, err := client.VolumeInspect(ctx, volumeName)
	if err != nil {
		return nil, err
	}
	return v.Labels, nil
}

// CreateVolume creates a Docker volume
func CreateVolume(volumeName string, driver string, driverOpts map[string]string, labels map[string]string) (vol dockerVolume.Volume, err error) {
	ctx, client := GetDockerClient()
	vol, err = client.VolumeCreate(ctx, dockerVolume.CreateOptions{Name: volumeName, Labels: labels, Driver: driver, DriverOpts: driverOpts})

	return vol, err
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
		// Lima specifies this as a named explicit IP address at this time
		// see https://github.com/lima-vm/lima/blob/master/docs/network.md#host-ip-19216852
		hostDockerInternal = "192.168.5.2"
		util.Debug("host.docker.internal=%s because running on Colima", hostDockerInternal)

	// Gitpod has Docker 20.10+ so the docker-compose has already gotten the host-gateway
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
		// If IDE is inside WSL2 then the normal Linux processing should work
		util.Debug("host.docker.internal='%s' because globalconfig.DdevGlobalConfig.XdebugIDELocation=%s", hostDockerInternal, globalconfig.XdebugIDELocationWSL2)
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():
		// Microsoft instructions for finding Windows IP address at
		// https://learn.microsoft.com/en-us/windows/wsl/networking#accessing-windows-networking-apps-from-linux-host-ip
		// If IDE is on Windows, we have to parse /etc/resolv.conf
		hostDockerInternal = wsl2GetWindowsHostIP()
		util.Debug("host.docker.internal='%s' because IsWSL2 and !IsDockerDesktop; received from ip -4 route show default", hostDockerInternal)

	// Docker on Linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop) defines host.docker.internal itself.
	case runtime.GOOS == "linux":
		// In Docker 20.10+, host.docker.internal is already taken care of by extra_hosts in docker-compose
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
// to get the actual addr in the case of Linux; still, Linux rarely
// is used with NFS. Returns "host.docker.internal" by default (not empty)
func GetNFSServerAddr() (string, error) {
	nfsAddr := "host.docker.internal"

	switch {
	case IsColima():
		// Lima specifies this as a named explicit IP address at this time
		// see https://github.com/lima-vm/lima/blob/master/docs/network.md#host-ip-19216852
		nfsAddr = "192.168.5.2"

	// Gitpod has Docker 20.10+ so the docker-compose has already gotten the host-gateway
	// However, NFS will never be used on Gitpod.
	case nodeps.IsGitpod():
		break
	case nodeps.IsCodespaces():
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():

		nfsAddr = wsl2GetWindowsHostIP()

	// Docker on Linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop) defines host.docker.internal itself.
	case runtime.GOOS == "linux":
		// Look up info from the bridge network
		// We can't use the Docker host because that's for inside the container,
		// and this is for setting up the network interface
		ctx, client := GetDockerClient()
		n, err := client.NetworkInspect(ctx, "bridge", dockerNetwork.InspectOptions{})
		if err != nil {
			return "", err
		}
		if len(n.IPAM.Config) > 0 {
			if n.IPAM.Config[0].Gateway != "" {
				nfsAddr = n.IPAM.Config[0].Gateway
			} else {
				util.Warning("Unable to determine Docker bridge gateway - no gateway")
			}
		}
	}

	return nfsAddr, nil
}

// 2024-04-13: The approach in wsl2ResolveConfNameserver no longer seems to be valid

// wsl2ResolvConfNameserver parses /etc/resolv.conf to get the nameserver,
// which is the only documented way to know how to connect to the host
// to connect to PhpStorm or other IDE listening there. Or for other apps.
//func wsl2ResolvConfNameserver() string {
//	if nodeps.IsWSL2() {
//		isAuto, err := fileutil.FgrepStringInFile("/etc/resolv.conf", "automatically generated by WSL")
//		if err != nil || !isAuto {
//			util.Warning("unable to determine WSL2 host.docker.internal because /etc/resolv.conf is not available or not auto-generated")
//			return ""
//		}
//		// We grepped it so no need to check error
//		etcResolv, _ := fileutil.ReadFileIntoString("/etc/resolv.conf")
//		util.Debug("resolv.conf=%s", etcResolv)
//
//		nameserverRegex := regexp.MustCompile(`nameserver *([0-9\.]*)`)
//		// nameserverRegex.ReplaceAllFunc([]byte(etcResolv), []byte(`$1`))
//		res := nameserverRegex.FindStringSubmatch(etcResolv)
//		if res == nil || len(res) != 2 {
//			util.Warning("unable to determine host.docker.internal from /etc/resolv.conf")
//			return ""
//		}
//		return res[1]
//	}
//	util.Warning("inappropriately using wsl2ResolvConfNameserver() but not on WSL2")
//	return ""
//}

// wsl2GetWindowsHostIP() uses ip -4 route show default to get the Windows IP address
// for use in determining host.docker.internal
func wsl2GetWindowsHostIP() string {
	// Get default route from WSL2
	out, err := ddevexec.RunHostCommand("ip", "-4", "route", "show", "default")

	if err != nil {
		util.Warning("Unable to run 'ip -4 route show default' to get Windows IP address")
		return ""
	}
	parts := strings.Split(out, " ")
	if len(parts) < 3 {
		util.Warning("Unable to parse output of 'ip -4 route show default', result was %v", parts)
		return ""
	}

	ip := parts[2]

	if parsedIP := net.ParseIP(ip); parsedIP == nil {
		util.Warning("Unable to validate IP address '%s' from 'ip -4 route show default'", ip)
		return ""
	}

	return ip
}

// RemoveImage removes an image with force
func RemoveImage(tag string) error {
	ctx, client := GetDockerClient()
	_, err := client.ImageInspect(ctx, tag)
	if err == nil {
		_, err = client.ImageRemove(ctx, tag, dockerImage.RemoveOptions{Force: true})

		if err == nil {
			util.Debug("Deleted Docker image %s", tag)
		} else {
			util.Warning("Unable to delete %s: %v", tag, err)
		}
	}
	return nil
}

// CopyIntoVolume copies a file or directory on the host into a Docker volume
// sourcePath is the host-side full path
// volumeName is the volume name to copy to
// targetSubdir is where to copy it to on the volume
// uid is the uid of the resulting files
// exclusion is a path to be excluded
// If destroyExisting the specified targetSubdir is removed and recreated
func CopyIntoVolume(sourcePath string, volumeName string, targetSubdir string, uid string, exclusion string, destroyExisting bool) error {
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

	var c = ""
	if destroyExisting {
		c = c + `rm -rf "` + targetSubdirFullPath + `"/{*,.*} && `
	}
	c = c + "mkdir -p " + targetSubdirFullPath + " && sleep infinity "

	containerID, _, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, map[string]string{"com.ddev.site-name": ""}, nil, nil)
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
	command := fmt.Sprintf("chown -R %s %s", uid, targetSubdirFullPath)
	stdout, stderr, err := Exec(containerID, command, "0")
	util.Debug("Exec %s stdout=%s, stderr=%s, err=%v", command, stdout, stderr, err)

	if err != nil {
		return err
	}
	track()
	return nil
}

// Exec does a simple docker exec, no frills, it executes the command
// with the specified uid (or defaults to root=0 if empty uid)
// Returns stdout, stderr, error
func Exec(containerID string, command string, uid string) (string, string, error) {
	ctx, client := GetDockerClient()

	if uid == "" {
		uid = "0"
	}
	execCreate, err := client.ContainerExecCreate(ctx, containerID, dockerContainer.ExecOptions{
		Cmd:          []string{"sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         uid,
	})
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	execAttach, err := client.ContainerExecAttach(ctx, execCreate.ID, dockerContainer.ExecStartOptions{
		Detach: false,
	})
	if err != nil {
		return "", "", err
	}
	defer execAttach.Close()

	_, err = stdcopy.StdCopy(&stdout, &stderr, execAttach.Reader)
	if err != nil {
		return "", "", err
	}

	info, err := client.ContainerExecInspect(ctx, execCreate.ID)
	if err != nil {
		return stdout.String(), stderr.String(), err
	}
	var execErr error
	if info.ExitCode != 0 {
		execErr = fmt.Errorf("command '%s' returned exit code %v", command, info.ExitCode)
	}

	return stdout.String(), stderr.String(), execErr
}

// CheckAvailableSpace outputs a warning if Docker space is low
func CheckAvailableSpace() {
	_, out, _ := RunSimpleContainer(ddevImages.GetWebImage(), "check-available-space-"+util.RandString(6), []string{"sh", "-c", `df / | awk '!/Mounted/ {print $4, $5;}'`}, []string{}, []string{}, []string{}, "", true, false, map[string]string{"com.ddev.site-name": ""}, nil, nil)
	out = strings.Trim(out, "% \r\n")
	parts := strings.Split(out, " ")
	if len(parts) != 2 {
		util.Warning("Unable to determine Docker space usage: %s", out)
		return
	}
	spacePercent, _ := strconv.Atoi(parts[1])
	spaceAbsolute, _ := strconv.Atoi(parts[0]) // Note that this is in KB

	if spaceAbsolute < nodeps.MinimumDockerSpaceWarning {
		util.Error("Your Docker install has only %d available disk space, less than %d warning level (%d%% used). Please increase disk image size.", spaceAbsolute, nodeps.MinimumDockerSpaceWarning, spacePercent)
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
	util.Debug("Downloading '%s' to '%s' ...", composeURL, destFile)

	_ = os.Remove(destFile)

	_ = os.MkdirAll(globalBinDir, 0777)
	err = util.DownloadFile(destFile, composeURL, "true" != os.Getenv("DDEV_NONINTERACTIVE"))
	if err != nil {
		return err
	}
	output.UserOut.Printf("Download complete.")

	// Remove the cached DockerComposeVersion
	globalconfig.DockerComposeVersion = ""

	err = util.Chmod(destFile, 0755)
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
	return "", fmt.Errorf("invalid docker-compose base version %s", v)
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
		return "", fmt.Errorf("only ARM64 and AMD64 architectures are supported for docker-compose v2, not %s", arch)
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
	ctx, client := GetDockerClient()
	info, err := client.Info(ctx)
	if err != nil {
		util.Warning("IsDockerDesktop(): Unable to get Docker info, err=%v", err)
		return false
	}
	if info.OperatingSystem == "Docker Desktop" {
		return true
	}
	return false
}

// IsColima detects if running on Colima
func IsColima() bool {
	ctx, client := GetDockerClient()
	info, err := client.Info(ctx)
	if err != nil {
		util.Warning("IsColima(): Unable to get Docker info, err=%v", err)
		return false
	}
	if strings.HasPrefix(info.Name, "colima") {
		return true
	}
	return false
}

// IsLima detects if running on lima
func IsLima() bool {
	ctx, client := GetDockerClient()
	info, err := client.Info(ctx)
	if err != nil {
		util.Warning("IsLima(): Unable to get Docker info, err=%v", err)
		return false
	}
	if info.Name != "lima-rancher-desktop" && strings.HasPrefix(info.Name, "lima") {
		return true
	}
	return false
}

// IsRancherDesktop detects if running on Rancher Desktop
func IsRancherDesktop() bool {
	ctx, client := GetDockerClient()
	info, err := client.Info(ctx)
	if err != nil {
		util.Warning("IsRancherDesktop(): Unable to get Docker info, err=%v", err)
		return false
	}
	if strings.HasPrefix(info.Name, "lima-rancher-desktop") {
		return true
	}
	return false
}

// IsOrbstack detects if running on Orbstack
func IsOrbstack() bool {
	ctx, client := GetDockerClient()
	info, err := client.Info(ctx)
	if err != nil {
		util.Warning("IsOrbstack(): Unable to get Docker info, err=%v", err)
		return false
	}
	if strings.HasPrefix(info.Name, "orbstack") {
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

	ctx, client := GetDockerClient()
	cid, err := FindContainerByName(containerName)
	if err != nil {
		return err
	}
	if cid == nil {
		return fmt.Errorf("copyIntoContainer unable to find a container named %s", containerName)
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

	err = client.CopyToContainer(ctx, cid.ID, dstPath, t, dockerContainer.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
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

	ctx, client := GetDockerClient()
	cid, err := FindContainerByName(containerName)
	if err != nil {
		return err
	}
	if cid == nil {
		return fmt.Errorf("copyFromContainer unable to find a container named %s", containerName)
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

	reader, _, err := client.CopyFromContainer(ctx, cid.ID, containerPath)
	if err != nil {
		return err
	}

	defer reader.Close()

	_, err = io.Copy(f, reader)
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

// DockerVersion is cached version of Docker provider engine
var DockerVersion = ""

// GetDockerVersion gets the cached or API-sourced version of Docker provider engine
func GetDockerVersion() (string, error) {
	if DockerVersion != "" {
		return DockerVersion, nil
	}
	ctx, client := GetDockerClient()
	if client == nil {
		return "", fmt.Errorf("unable to get Docker provider engine version: Docker client is nil")
	}

	serverVersion, err := client.ServerVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get Docker provider engine version: %v", err)
	}

	DockerVersion = serverVersion.Version

	return DockerVersion, nil
}

// DockerAPIVersion is cached API version of Docker provider engine
// See https://docs.docker.com/engine/api/#api-version-matrix
var DockerAPIVersion = ""

// GetDockerAPIVersion gets the cached or API-sourced API version of Docker provider engine
func GetDockerAPIVersion() (string, error) {
	if DockerAPIVersion != "" {
		return DockerAPIVersion, nil
	}
	ctx, client := GetDockerClient()
	if client == nil {
		return "", fmt.Errorf("unable to get Docker provider engine API version: Docker client is nil")
	}

	serverVersion, err := client.ServerVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get Docker provider engine API version: %v", err)
	}

	DockerAPIVersion = serverVersion.APIVersion

	return DockerAPIVersion, nil
}

// DockerComposeVersionConstraint is the versions allowed for ddev
// REMEMBER TO CHANGE docs/ddev-installation.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/ddev/ddev/pull/738 and regression https://github.com/ddev/ddev/issues/1431
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

// GetContainerNames takes an array of Container
// and returns an array of strings with container names.
// Use removePrefix to get short container names.
func GetContainerNames(containers []dockerContainer.Summary, excludeContainerNames []string, removePrefix string) []string {
	var names []string
	for _, container := range containers {
		if len(container.Names) == 0 {
			continue
		}
		name := container.Names[0][1:] // Trimming the leading '/' from the container name
		if slices.Contains(excludeContainerNames, name) {
			continue
		}
		if removePrefix != "" {
			name = strings.TrimPrefix(name, removePrefix)
		}
		names = append(names, name)
	}
	return names
}

// IsErrNotFound returns true if the error is a NotFound error, which is returned
// by the API when some object is not found. It is an alias for [errdefs.IsNotFound].
// Used as a wrapper to avoid direct import for docker client.
func IsErrNotFound(err error) bool {
	return dockerClient.IsErrNotFound(err)
}

// CanRunWithoutDocker returns true if the command or flag can run without Docker.
func CanRunWithoutDocker() bool {
	if len(os.Args) < 2 {
		return true
	}
	// Check the first arg
	if slices.Contains([]string{"-v", "--version", "-h", "--help", "help", "hostname"}, os.Args[1]) {
		return true
	}
	// Check the last arg
	if slices.Contains([]string{"-h", "--help"}, os.Args[len(os.Args)-1]) {
		// Some commands don't support Cobra help, because they are wrappers
		if slices.Contains([]string{"composer"}, os.Args[1]) {
			return false
		}
		return true
	}
	return false
}

// ValidatePort checks that the given port is valid (in range 1-65535)
func ValidatePort(port interface{}) error {
	var dockerPort int
	switch v := port.(type) {
	case int:
		dockerPort = v
	case string:
		var err error
		dockerPort, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid port: %v", port)
		}
	default:
		return fmt.Errorf("unsupported port type: %T", port)
	}
	if dockerPort < 1 || dockerPort > 65535 {
		return fmt.Errorf("invalid port: %v", port)
	}
	return nil
}
