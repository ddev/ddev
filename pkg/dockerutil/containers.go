package dockerutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/moby/term"
)

// NoHealthCheck is a HealthConfig that disables any existing healthcheck when
// running a container. Used by RunSimpleContainer
// See https://pkg.go.dev/github.com/moby/docker-image-spec/specs-go/v1#HealthcheckConfig
var NoHealthCheck = container.HealthConfig{
	Test: []string{"NONE"}, // Disables any existing health check
}

// InspectContainer returns the full result of inspection
func InspectContainer(name string) (container.InspectResponse, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return container.InspectResponse{}, err
	}

	c, err := FindContainerByName(name)
	if err != nil || c == nil {
		return container.InspectResponse{}, err
	}
	x, err := client.ContainerInspect(ctx, c.ID)
	return x, err
}

// FindContainerByName takes a container name and returns the container
// If container is not found, returns nil with no error
func FindContainerByName(name string) (*container.Summary, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	containers, err := client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: name}),
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
	c, err := FindContainerByName(name)
	if err != nil || c == nil {
		return "doesnotexist", fmt.Errorf("container %s does not exist", name)
	}
	if c.State == "running" {
		return c.State, nil
	}
	return c.State, fmt.Errorf("container %s is in state=%s so can't be accessed", name, c.State)
}

// FindContainerByLabels takes a map of label names and values and returns any Docker containers which match all labels.
func FindContainerByLabels(labels map[string]string) (*container.Summary, error) {
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
func GetDockerContainers(allContainers bool) ([]container.Summary, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := client.ContainerList(ctx, container.ListOptions{All: allContainers})
	return containers, err
}

// FindContainersByLabels takes a map of label names and values and returns any Docker containers which match all labels.
// Explanation of the query:
// * docs: https://docs.docker.com/engine/api/v1.23/
// * Stack Overflow: https://stackoverflow.com/questions/28054203/docker-remote-api-filter-exited
func FindContainersByLabels(labels map[string]string) ([]container.Summary, error) {
	if len(labels) < 1 {
		return nil, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := filters.NewArgs()
	for k, v := range labels {
		label := fmt.Sprintf("%s=%s", k, v)
		// If no value is specified, filter any value by the key.
		if v == "" {
			label = k
		}
		filterList.Add("label", label)
	}

	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := client.ContainerList(ctx, container.ListOptions{
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
func FindContainersWithLabel(label string) ([]container.Summary, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "label", Value: label}),
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
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
			c, err := FindContainerByLabels(labels)
			if err == nil && c != nil {
				health, _ := GetContainerHealth(c)
				if health != "healthy" {
					name, suggestedCommand := getSuggestedCommandForContainerLog(c, waittime)
					desc = desc + fmt.Sprintf(" %s:%s\n%s", name, health, suggestedCommand)
				}
			}
			return "", fmt.Errorf("health check timed out after %v: labels %v timed out without becoming healthy, status=%v, detail=%s ", durationWait, labels, status, desc)

		case <-tickChan.C:
			c, err := FindContainerByLabels(labels)
			if err != nil || c == nil {
				return "", fmt.Errorf("failed to query container labels=%v: %v", labels, err)
			}
			health, logOutput := GetContainerHealth(c)

			switch health {
			case "healthy":
				return logOutput, nil
			case "unhealthy":
				name, suggestedCommand := getSuggestedCommandForContainerLog(c, 0)
				return logOutput, fmt.Errorf("%s container is unhealthy, log=%s\n%s", name, logOutput, suggestedCommand)
			case "exited":
				name, suggestedCommand := getSuggestedCommandForContainerLog(c, 0)
				return logOutput, fmt.Errorf("%s container exited,\n%s", name, suggestedCommand)
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
						name, suggestedCommand := getSuggestedCommandForContainerLog(&c, waittime)
						desc = desc + fmt.Sprintf(" %s:%s\n%s", name, health, suggestedCommand)
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
			for _, c := range containers {
				health, logOutput := GetContainerHealth(&c)

				switch health {
				case "healthy":
					continue
				case "unhealthy":
					name, suggestedCommand := getSuggestedCommandForContainerLog(&c, 0)
					return fmt.Errorf("%s container is unhealthy, log=%s\n%s", name, logOutput, suggestedCommand)
				case "exited":
					name, suggestedCommand := getSuggestedCommandForContainerLog(&c, 0)
					return fmt.Errorf("%s container exited.\n%s", name, suggestedCommand)
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

// getSuggestedCommandForContainerLog returns a command that can be used to find out what is wrong with a container
func getSuggestedCommandForContainerLog(container *container.Summary, timeout int) (string, string) {
	var suggestedCommands []string
	service := container.Labels["com.docker.compose.service"]
	if service != "" && service != "ddev-router" && service != "ddev-ssh-agent" {
		suggestedCommands = append(suggestedCommands, fmt.Sprintf("ddev logs -s %s", service))
	}
	name := ContainerName(container)
	suggestedCommands = append(suggestedCommands, fmt.Sprintf("docker logs %s", name), fmt.Sprintf("docker inspect --format \"{{ json .State.Health }}\" %s | docker run -i --rm ddev/ddev-utilities jq -r", name))
	troubleshootingCommand, _ := util.ArrayToReadableOutput(suggestedCommands)
	suggestedCommand := "\nTroubleshoot this with these commands:\n" + troubleshootingCommand
	if timeout > 0 && service != "ddev-router" && service != "ddev-ssh-agent" {
		timeoutNote := "\nIf your internet connection is slow, consider increasing the timeout by running this:\n"
		timeoutCommand, _ := util.ArrayToReadableOutput([]string{fmt.Sprintf("ddev config --default-container-timeout=%d && ddev restart", timeout*2)})
		suggestedCommand = suggestedCommand + timeoutNote + timeoutCommand
	}
	return name, suggestedCommand
}

// ContainerName returns the container's human-readable name.
func ContainerName(container *container.Summary) string {
	if len(container.Names) == 0 {
		return container.ID
	}
	return container.Names[0][1:]
}

// GetContainerHealth retrieves the health status of a given container.
// returns status, most-recent-log
func GetContainerHealth(container *container.Summary) (string, string) {
	if container == nil {
		return "no container", ""
	}

	// If the container is not running, then return exited as the health.
	// "exited" means stopped.
	if container.State == "exited" || container.State == "restarting" {
		return container.State, ""
	}

	ctx, client, err := GetDockerClient()
	if err != nil {
		return "", ""
	}
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

// GetAppContainers retrieves docker containers for a given sitename.
func GetAppContainers(sitename string) ([]container.Summary, error) {
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
func GetContainerEnv(key string, container container.Summary) string {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return ""
	}
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

// GetPublishedPort returns the published port for a given private port.
func GetPublishedPort(privatePort uint16, container container.Summary) int {
	for _, port := range container.Ports {
		if port.PrivatePort == privatePort {
			return int(port.PublicPort)
		}
	}
	return 0
}

// RunSimpleContainer runs a container (non-daemonized) and captures the stdout/stderr.
// It will block, so not to be run on a container whose entrypoint or cmd might hang or run too long.
// This should be the equivalent of something like
// docker run -t -u '%s:%s' -e SNAPSHOT_NAME='%s' -v '%s:/mnt/ddev_config' -v '%s:/var/lib/mysql' --no-healthcheck --rm --entrypoint=/migrate_file_to_volume.sh %s:%s
// Example code from https://gist.github.com/fsouza/b0bf3043827f8e39c4589e88cec067d8
// Default behavior is to use the image's healthcheck (healthConfig == nil)
// When passed a pointer to HealthConfig (often &dockerutils.NoHealthCheck) it can turn off healthcheck,
// or it can replace it or have other behaviors, see
// https://pkg.go.dev/github.com/moby/docker-image-spec/specs-go/v1#HealthcheckConfig
// Returns containerID, output, error
func RunSimpleContainer(image string, name string, cmd []string, entrypoint []string, env []string, binds []string, uid string, removeContainerAfterRun bool, detach bool, labels map[string]string, portBindings nat.PortMap, healthConfig *container.HealthConfig) (containerID string, output string, returnErr error) {
	config := &container.Config{
		Image:       image,
		Cmd:         cmd,
		Env:         env,
		User:        uid,
		Labels:      labels,
		Entrypoint:  entrypoint,
		Healthcheck: healthConfig,
	}
	hostConfig := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
	}
	return RunSimpleContainerExtended(name, config, hostConfig, removeContainerAfterRun, detach)
}

// RunSimpleContainerExtended runs a container (non-daemonized) and captures the stdout/stderr.
// Accepts any config and hostConfig. If stdin is provided, enables interactive mode with stdin forwarding.
func RunSimpleContainerExtended(name string, config *container.Config, hostConfig *container.HostConfig, removeContainerAfterRun bool, detach bool) (containerID string, output string, returnErr error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return "", "", err
	}

	image := config.Image
	if image == "" {
		return "", "", fmt.Errorf("RunSimpleContainerExtended requires config.Image to be set")
	}

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

	config.AttachStderr = true
	config.AttachStdout = true

	if config.AttachStdin {
		config.AttachStdin = true
		config.OpenStdin = true
		config.Tty = term.IsTerminal(os.Stdin.Fd())
	}

	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}
	// Assign a default label so this container can be removed with 'ddev poweroff'
	if _, exists := config.Labels["com.ddev.site-name"]; !exists {
		config.Labels["com.ddev.site-name"] = ""
	}

	// Set up host.docker.internal based on DDEV's standard approach
	hostDockerInternal := GetHostDockerInternal()
	if hostDockerInternal.ExtraHost != "" && !slices.Contains(hostConfig.ExtraHosts, "host.docker.internal:"+hostDockerInternal.ExtraHost) {
		hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, "host.docker.internal:"+hostDockerInternal.ExtraHost)
	}

	c, err := client.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		return "", "", fmt.Errorf("failed to create/start Docker container %v (%v, %v): %v", name, config, hostConfig, err)
	}

	if removeContainerAfterRun {
		// nolint: errcheck
		defer RemoveContainer(c.ID)
	}

	var hijackedResp *types.HijackedResponse
	var outputDone chan struct{}

	if config.AttachStdin {
		// Interactive mode with stdin - use attach for real-time I/O
		attachOptions := container.AttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		}

		resp, err := client.ContainerAttach(ctx, c.ID, attachOptions)
		if err != nil {
			return c.ID, "", fmt.Errorf("failed to attach to container: %v", err)
		}
		hijackedResp = &resp
		defer hijackedResp.Close()

		restoreTerminal, err := setupRawTerminal()
		if err != nil {
			return c.ID, "", err
		}
		defer restoreTerminal()

		// Initialize synchronization channel for output completion
		outputDone = make(chan struct{})

		// Forward output from container to stdout
		go func() {
			_, _ = io.Copy(os.Stdout, hijackedResp.Reader)
			if outputDone != nil {
				close(outputDone)
			}
		}()

		// Forward input from stdin to container
		go func() {
			_, _ = io.Copy(hijackedResp.Conn, os.Stdin)
		}()
	}

	if err := client.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
		return c.ID, "", fmt.Errorf("failed to StartContainer: %v", err)
	}

	exitCode := 0

	if !detach {
		waitChan, errChan := client.ContainerWait(ctx, c.ID, container.WaitConditionNotRunning)
		select {
		case status := <-waitChan:
			exitCode = int(status.StatusCode)
		case err := <-errChan:
			return c.ID, "", fmt.Errorf("failed to ContainerWait: %v", err)
		}

		// For interactive containers, wait for I/O forwarding to complete
		if config.AttachStdin {
			// Close hijacked connection to signal EOF to goroutines
			hijackedResp.Close()

			// Wait for output forwarding to complete (input may still be running)
			if outputDone != nil {
				<-outputDone
			}
		}
	}

	var exitErr error

	if exitCode != 0 {
		exitErr = fmt.Errorf("container run failed with exit code %d", exitCode)
	}

	// Don't capture logs if we're attaching stdin, as it will block stdout
	if config.AttachStdin {
		return c.ID, "", exitErr
	}

	// Get logs so we can report them
	options := container.LogsOptions{ShowStdout: true, ShowStderr: true}
	rc, err := client.ContainerLogs(ctx, c.ID, options)
	if err != nil {
		return c.ID, "", fmt.Errorf("failed to get container logs: %v", err)
	}
	defer rc.Close()

	var stdout bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stdout, rc)
	if err != nil {
		return c.ID, "", fmt.Errorf("failed to copy container logs: %v", err)
	}

	return c.ID, stdout.String(), exitErr
}

// RemoveContainer stops and removes a container
func RemoveContainer(id string) error {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}

	err = client.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	return err
}

// RemoveContainersByLabels removes all containers that match a set of labels
func RemoveContainersByLabels(labels map[string]string) error {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
	containers, err := FindContainersByLabels(labels)
	if err != nil {
		return err
	}
	if containers == nil {
		return nil
	}
	for _, c := range containers {
		err = client.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBoundHostPorts takes a container pointer and returns an array
// of exposed ports (and error)
func GetBoundHostPorts(containerID string) ([]string, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
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
	var ports []string
	for k := range portMap {
		ports = append(ports, k)
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i] < ports[j]
	})
	return ports, nil
}

// Exec does a simple docker exec, no frills, it executes the command
// with the specified uid (or defaults to root=0 if empty uid)
// Returns stdout, stderr, error
func Exec(containerID string, command string, uid string) (string, string, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return "", "", err
	}

	if uid == "" {
		uid = "0"
	}
	execCreate, err := client.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         uid,
	})
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	execAttach, err := client.ContainerExecAttach(ctx, execCreate.ID, container.ExecStartOptions{
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

	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
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

	err = client.CopyToContainer(ctx, cid.ID, dstPath, t, container.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
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

	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
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

// GetContainerNames takes an array of Container
// and returns an array of strings with container names.
// Use removePrefix to get short container names.
func GetContainerNames(containers []container.Summary, excludeContainerNames []string, removePrefix string) []string {
	var names []string
	for _, c := range containers {
		if len(c.Names) == 0 {
			continue
		}
		name := c.Names[0][1:] // Trimming the leading '/' from the container name
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

// TruncateID returns a shorthand version of a string identifier for convenience.
// This is a copy from https://github.com/moby/moby/blob/master/client/pkg/stringid/stringid.go
func TruncateID(id string) string {
	if i := strings.IndexRune(id, ':'); i >= 0 {
		id = id[i+1:]
	}
	shortLen := 12
	if len(id) > shortLen {
		id = id[:shortLen]
	}
	return id
}

// setupRawTerminal sets the terminal to raw mode for TTY containers.
// Returns a restore function that should be called to restore terminal state.
func setupRawTerminal() (restore func(), err error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return func() {}, nil
	}

	stdinState, err := term.SetRawTerminal(os.Stdin.Fd())
	if err != nil {
		return nil, fmt.Errorf("failed to set stdin to raw mode: %v", err)
	}

	stdoutState, err := term.SetRawTerminalOutput(os.Stdout.Fd())
	if err != nil {
		_ = term.RestoreTerminal(os.Stdin.Fd(), stdinState)
		return nil, fmt.Errorf("failed to set stdout to raw mode: %v", err)
	}

	restore = func() {
		if stdinState != nil {
			_ = term.RestoreTerminal(os.Stdin.Fd(), stdinState)
		}
		if stdoutState != nil {
			_ = term.RestoreTerminal(os.Stdout.Fd(), stdoutState)
		}
	}

	return restore, nil
}
