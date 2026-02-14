package dockerutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ddev/ddev/pkg/archive"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/moby/term"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// NoHealthCheck is a HealthConfig that disables any existing healthcheck when
// running a container. Used by RunSimpleContainer
// See https://pkg.go.dev/github.com/moby/docker-image-spec/specs-go/v1#HealthcheckConfig
var NoHealthCheck = container.HealthConfig{
	Test: []string{"NONE"}, // Disables any existing health check
}

// containerUser holds the UID, GID, and username used to run containers
type containerUser struct {
	uidStr   string
	gidStr   string
	username string
}

var (
	// sContainerUser is the singleton instance of containerUser
	sContainerUser *containerUser
	// sContainerUserOnce ensures sContainerUser is initialized only once
	sContainerUserOnce sync.Once
)

// sanitizeUsername converts a username to be safe for Linux containers.
// Linux usernames can only contain: a-z, 0-9, _, -
// and must start with a letter.
func sanitizeUsername(rawUsername string) string {
	username := rawUsername

	// Normalize unicode characters (remove diacritics)
	// Per https://stackoverflow.com/a/65981868/215713
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	username, _, _ = transform.String(t, username)

	// Handle Windows domain\user format - extract username after backslash
	if idx := strings.LastIndex(username, `\`); idx >= 0 {
		username = username[idx+1:]
	}

	// Lowercase and remove all invalid characters
	// Linux usernames can only contain: a-z, 0-9, _, -
	username = strings.ToLower(username)
	username = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1 // Remove character
	}, username)

	// Ensure username starts with a letter (prepend 'a' if not)
	// Example issue: username="310822" in https://github.com/ddev/ddev/issues/3187
	if len(username) == 0 || !nodeps.IsLetter(string(username[0])) {
		username = "a" + username
	}

	return username
}

// GetContainerUser returns the uid, gid, and username used to run most containers
func GetContainerUser() (uidStr string, gidStr string, username string) {
	sContainerUserOnce.Do(func() {
		// Default fallback values if we can't determine the user
		uidStr = "1000"
		gidStr = "1000"
		username = "ddev"

		curUser, err := user.Current()
		if err != nil {
			// Use fallback values and warn
			util.Warning("Unable to determine current user (UID, GID, username), using fallback uid=%s gid=%s username=%s: %v", uidStr, gidStr, username, err)
		} else {
			// Use actual user values
			uidStr = curUser.Uid
			gidStr = curUser.Gid
			username = curUser.Username

			// Sanitize username for safe use in Linux containers
			// Example problem usernames: "André Kraus", "Mück", "DOMAIN\user", "user@example.com"
			// See https://stackoverflow.com/questions/64933879
			username = sanitizeUsername(username)
		}

		// Windows user IDs are non-numeric,
		// so we have to run as arbitrary user 1000. We may have a host uidStr/gidStr greater in other contexts,
		// 1000 seems not to cause file permissions issues at least on docker-for-windows.
		if nodeps.IsWindows() {
			uidStr = "1000"
			gidStr = "1000"
		}
		sContainerUser = &containerUser{
			uidStr:   uidStr,
			gidStr:   gidStr,
			username: username,
		}
	})

	return sContainerUser.uidStr, sContainerUser.gidStr, sContainerUser.username
}

// InspectContainer returns the full result of inspection
func InspectContainer(name string) (container.InspectResponse, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return container.InspectResponse{}, err
	}

	c, err := FindContainerByName(name)
	if err != nil || c == nil {
		return container.InspectResponse{}, err
	}
	x, err := apiClient.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return x.Container, err
}

// FindContainerByName takes a container name and returns the container
// If container is not found, returns nil with no error
func FindContainerByName(name string) (*container.Summary, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("name", name),
	})
	if err != nil {
		return nil, err
	}
	if len(containers.Items) == 0 {
		return nil, nil
	}

	// ListContainers can return partial matches. Make sure we only match the exact one
	// we're after.
	for _, c := range containers.Items {
		if len(c.Names) > 0 && c.Names[0] == "/"+name {
			return &c, nil
		}
	}
	return nil, nil
}

// GetContainerStateByName returns container state for the named container
func GetContainerStateByName(name string) (container.ContainerState, error) {
	c, err := FindContainerByName(name)
	if err != nil || c == nil {
		return "doesnotexist", fmt.Errorf("container %s does not exist", name)
	}
	if c.State == container.StateRunning {
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{All: allContainers})
	if err != nil {
		return nil, err
	}
	return containers.Items, err
}

// FindContainersByLabels takes a map of label names and values and returns any Docker containers which match all labels.
// Explanation of the query:
// * docs: https://docs.docker.com/engine/api/v1.23/
// * Stack Overflow: https://stackoverflow.com/questions/28054203/docker-remote-api-filter-exited
func FindContainersByLabels(labels map[string]string) ([]container.Summary, error) {
	if len(labels) < 1 {
		return nil, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := client.Filters{}
	for k, v := range labels {
		label := fmt.Sprintf("%s=%s", k, v)
		// If no value is specified, filter any value by the key.
		if v == "" {
			label = k
		}
		filterList = filterList.Add("label", label)
	}

	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterList,
	})
	if err != nil {
		return nil, err
	}
	return containers.Items, nil
}

// FindContainersWithLabel returns all containers with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindContainersWithLabel(label string) ([]container.Summary, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("label", label),
	})
	if err != nil {
		return nil, err
	}

	return containers.Items, nil
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
	lastStatus := ""
	startTime := time.Now()
	lastLogTime := startTime

	for {
		select {
		case <-timeoutChan.C:
			_ = timeoutChan.Stop()
			desc := ""
			c, err := FindContainerByLabels(labels)
			if err == nil && c != nil {
				health, _ := GetContainerHealth(c)
				if health != string(container.Healthy) {
					name, suggestedCommand := getSuggestedCommandForContainerLog(c, waittime)
					desc = desc + fmt.Sprintf(" %s:%s\n%s", name, health, suggestedCommand)
				}
			}
			return "", fmt.Errorf("health check timed out after %v: labels %v timed out without becoming healthy, status=%v, detail=%s ", durationWait, labels, status, desc)

		case <-tickChan.C:
			c, err := FindContainerByLabels(labels)
			cName := ""
			if err != nil || c == nil {
				return "", fmt.Errorf("failed to query container %s labels=%v: %v", cName, labels, err)
			}
			if len(c.Names) > 0 {
				cName = strings.TrimPrefix(c.Names[0], "/")
			}
			health, logOutput := GetContainerHealth(c)

			// Log status changes and periodic updates under DDEV_DEBUG
			elapsed := time.Since(startTime).Round(time.Millisecond)
			if health != lastStatus {
				util.Debug("ContainerWait: %s status change: '%s' after %v", cName, health, elapsed)
				lastStatus = health
				lastLogTime = time.Now()
			} else if time.Since(lastLogTime) >= 5*time.Second {
				util.Debug("ContainerWait: still waiting for %s, status='%s' after %v", cName, health, elapsed)
				lastLogTime = time.Now()
			}

			switch health {
			case string(container.Healthy):
				return logOutput, nil
			case string(container.Unhealthy):
				name, suggestedCommand := getSuggestedCommandForContainerLog(c, 0)
				return logOutput, fmt.Errorf("%s container is unhealthy, log=%s\n%s", name, logOutput, suggestedCommand)
			case string(container.StateExited):
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
// filterServices optionally limits which containers are checked by their
// "com.docker.compose.service" label. When empty, all containers matching
// labels are checked.
// Returns error if not all containers become "healthy" before the timeout.
func ContainersWait(waittime int, labels map[string]string, filterServices ...string) error {
	timeoutChan := time.After(time.Duration(waittime) * time.Second)
	tickChan := time.NewTicker(500 * time.Millisecond)
	defer tickChan.Stop()

	status := ""
	lastStatus := ""
	startTime := time.Now()
	lastLogTime := startTime

	for {
		select {
		case <-timeoutChan:
			desc := ""
			containers, err := FindContainersByLabels(labels)
			if err == nil && containers != nil {
				for _, c := range containers {
					if len(filterServices) > 0 && !slices.Contains(filterServices, c.Labels["com.docker.compose.service"]) {
						continue
					}
					health, _ := GetContainerHealth(&c)
					if health != string(container.Healthy) {
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
			healthyCount := 0
			totalCount := 0
			for _, c := range containers {
				if len(filterServices) > 0 && !slices.Contains(filterServices, c.Labels["com.docker.compose.service"]) {
					continue
				}
				totalCount++
				health, logOutput := GetContainerHealth(&c)

				switch health {
				case string(container.Healthy):
					healthyCount++
					continue
				case string(container.Unhealthy):
					name, suggestedCommand := getSuggestedCommandForContainerLog(&c, 0)
					return fmt.Errorf("%s container is unhealthy, log=%s\n%s", name, logOutput, suggestedCommand)
				case string(container.StateExited):
					name, suggestedCommand := getSuggestedCommandForContainerLog(&c, 0)
					return fmt.Errorf("%s container exited.\n%s", name, suggestedCommand)
				default:
					allHealthy = false
				}
			}

			if totalCount == 0 {
				allHealthy = false
			}

			// Log status changes and periodic updates under DDEV_DEBUG
			currentStatus := fmt.Sprintf("%d/%d healthy", healthyCount, totalCount)
			elapsed := time.Since(startTime).Round(time.Millisecond)
			if currentStatus != lastStatus {
				util.Debug("ContainersWait: status changed to '%s' after %v", currentStatus, elapsed)
				lastStatus = currentStatus
				lastLogTime = time.Now()
			} else if time.Since(lastLogTime) >= 5*time.Second {
				util.Debug("ContainersWait: still waiting, status='%s' after %v", currentStatus, elapsed)
				lastLogTime = time.Now()
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
func getSuggestedCommandForContainerLog(c *container.Summary, timeout int) (string, string) {
	var suggestedCommands []string
	service := c.Labels["com.docker.compose.service"]
	if service != "" && service != "ddev-router" && service != "ddev-ssh-agent" {
		suggestedCommands = append(suggestedCommands, fmt.Sprintf("ddev logs -s %s", service))
	}
	name := ContainerName(c)
	suggestedCommands = append(suggestedCommands, fmt.Sprintf("docker logs %s", name), fmt.Sprintf("docker inspect --format \"{{ json .State.Health }}\" %s | docker run -i --rm ddev/ddev-utilities jq -r", name))
	troubleshootingCommand, _ := util.ArrayToReadableOutput(suggestedCommands)
	suggestedCommand := "\nTroubleshoot this with these commands:\n" + troubleshootingCommand
	if timeout > 0 && service != "ddev-router" && service != "ddev-ssh-agent" {
		timeoutNote := "\nIf your internet connection is slow, consider increasing the timeout by running this:\n"
		timeoutCommand, _ := util.ArrayToReadableOutput([]string{fmt.Sprintf("ddev config --default-container-timeout=%d && ddev restart", timeout*2)})
		suggestedCommand = suggestedCommand + timeoutNote + timeoutCommand
	}
	if globalconfig.DdevDebug {
		ctx, apiClient, err := GetDockerClient()
		if err == nil {
			var stdout bytes.Buffer
			logOpts := client.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     false,
				Timestamps: false,
			}
			rc, err := apiClient.ContainerLogs(ctx, c.ID, logOpts)
			if err != nil {
				util.Warning("Unable to capture logs from %s container: %v", name, err)
			} else {
				defer rc.Close()
				_, err = stdcopy.StdCopy(&stdout, &stdout, rc)
				if err != nil {
					util.Warning("Unable to copy logs from %s container: %v", name, err)
				}
				util.Debug("Logs from failed %s container:\n%s\n", name, strings.TrimSpace(stdout.String()))
			}
			_, logOutput := GetContainerHealth(c)
			util.Debug("Health log from failed %s container:\n%s\n", name, strings.TrimSpace(logOutput))
		}
	}
	return name, suggestedCommand
}

// ContainerName returns the container's human-readable name.
func ContainerName(c *container.Summary) string {
	if len(c.Names) == 0 {
		return c.ID
	}
	return c.Names[0][1:]
}

// GetContainerHealth retrieves the health status of a given container.
// returns status, most-recent-log
// The container is only considered "healthy" if it's also "running", contrary to Docker's normal usage
func GetContainerHealth(c *container.Summary) (string, string) {
	if c == nil {
		return "no container", ""
	}

	// If the container is not running, then return exited as the health.
	// "exited" means stopped.
	cState := string(c.State)
	if cState == string(container.StateExited) || cState == string(container.StateRestarting) {
		return cState, ""
	}

	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return "", ""
	}
	inspect, err := apiClient.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
	if err != nil {
		output.UserOut.Warnf("Error getting container to inspect: %v", err)
		return "", ""
	}

	logOutput := ""
	status := ""
	if inspect.Container.State.Health != nil {
		status = string(inspect.Container.State.Health.Status)
	}
	// The last log is the most recent
	if status != "" {
		numLogs := len(inspect.Container.State.Health.Log)
		if numLogs > 0 {
			logOutput = fmt.Sprintf("%v", inspect.Container.State.Health.Log[numLogs-1].Output)
		}
		// A container can't be healthy if it's not running.
		// Docker/Podman may cache the last health status even after state changes.
		if inspect.Container.State.Status != container.StateRunning {
			// Return actual state for known non-running states
			switch inspect.Container.State.Status {
			case container.StateExited, container.StateRestarting:
				status = string(inspect.Container.State.Status)
			default:
				// For other non-running states, override cached health to unhealthy
				status = string(container.Unhealthy)
			}
		}
	} else {
		// Some containers may not have a healthcheck. In that case
		// we use State to determine health
		switch inspect.Container.State.Status {
		case container.StateRunning:
			status = string(container.Healthy)
		case container.StateExited:
			status = string(container.StateExited)
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
func GetContainerEnv(key string, c container.Summary) string {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return ""
	}
	inspect, err := apiClient.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})

	if err == nil {
		envVars := inspect.Container.Config.Env

		for _, env := range envVars {
			if strings.HasPrefix(env, key) {
				return strings.TrimPrefix(env, key+"=")
			}
		}
	}
	return ""
}

// GetPublishedPort returns the published port for a given private port.
func GetPublishedPort(privatePort uint16, c container.Summary) int {
	for _, port := range c.Ports {
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
func RunSimpleContainer(image string, name string, cmd []string, entrypoint []string, env []string, binds []string, uid string, removeContainerAfterRun bool, detach bool, labels map[string]string, portBindings network.PortMap, healthConfig *container.HealthConfig) (containerID string, out string, returnErr error) {
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
func RunSimpleContainerExtended(name string, config *container.Config, hostConfig *container.HostConfig, removeContainerAfterRun bool, detach bool) (containerID string, out string, returnErr error) {
	ctx, apiClient, err := GetDockerClient()
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

	// Empty user means root
	if config.User == "" {
		config.User = "0"
	}

	if IsPodman() {
		if config.Healthcheck == nil {
			// Podman doesn't recognize HEALTHCHECK from Dockerfile
			// https://github.com/containers/podman/issues/18904
			// We can set it explicitly
			if strings.HasPrefix(config.Image, ddevImages.GetWebImage()) {
				// HEALTHCHECK from containers/ddev-webserver/Dockerfile
				config.Healthcheck = &container.HealthConfig{
					Test:        []string{"CMD-SHELL", "/healthcheck.sh"},
					Interval:    1 * time.Second,
					Timeout:     120 * time.Second,
					StartPeriod: 120 * time.Second,
					Retries:     120,
				}
			}
		}
		if config.User == "0" {
			// For containers that use CopyIntoContainer or run "chown", set UsernsMode,
			// otherwise file ownership inside the container will be incorrect
			if usernsMode, exists := config.Labels["com.ddev.userns"]; exists {
				hostConfig.UsernsMode = container.UsernsMode(usernsMode)
			}
		} else {
			// Always use "keep-id" for non-root users
			hostConfig.UsernsMode = "keep-id"
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

	c, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{Config: config, HostConfig: hostConfig, Name: name})
	if err != nil {
		return "", "", fmt.Errorf("failed to create/start Docker container %v (%v, %v): %v", name, config, hostConfig, err)
	}

	if removeContainerAfterRun {
		// nolint: errcheck
		defer RemoveContainer(c.ID)
	}

	var hijackedResp *client.HijackedResponse
	var outputDone chan struct{}

	if config.AttachStdin {
		// Interactive mode with stdin - use attach for real-time I/O
		attachOptions := client.ContainerAttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		}

		resp, err := apiClient.ContainerAttach(ctx, c.ID, attachOptions)
		if err != nil {
			return c.ID, "", fmt.Errorf("failed to attach to container: %v", err)
		}
		hijackedResp = &resp.HijackedResponse
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
			if output.JSONOutput {
				util.Warning("Ignoring all output from container in JSON mode with stdin attached")
				buf := new(bytes.Buffer)
				_, _ = buf.ReadFrom(hijackedResp.Reader)
				text := strings.ReplaceAll(buf.String(), "\r\n", "\n")
				text = strings.ReplaceAll(text, "\r", "\n")
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					if line != "" {
						output.UserOut.Println(line)
					}
				}
			} else {
				_, _ = io.Copy(os.Stdout, hijackedResp.Reader)
			}
			if outputDone != nil {
				close(outputDone)
			}
		}()

		// Forward input from stdin to container
		go func() {
			_, _ = io.Copy(hijackedResp.Conn, os.Stdin)
		}()
	}

	if _, err := apiClient.ContainerStart(ctx, c.ID, client.ContainerStartOptions{}); err != nil {
		return c.ID, "", fmt.Errorf("failed to StartContainer: %v", err)
	}

	exitCode := 0

	if !detach {
		waitResult := apiClient.ContainerWait(ctx, c.ID, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
		select {
		case status := <-waitResult.Result:
			exitCode = int(status.StatusCode)
		case err := <-waitResult.Error:
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
	options := client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	rc, err := apiClient.ContainerLogs(ctx, c.ID, options)
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return err
	}

	_, err = apiClient.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	return err
}

// RemoveContainersByLabels removes all containers that match a set of labels
func RemoveContainersByLabels(labels map[string]string) error {
	ctx, apiClient, err := GetDockerClient()
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
		_, err = apiClient.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBoundHostPorts takes a container pointer and returns an array
// of exposed ports (and error)
func GetBoundHostPorts(containerID string) ([]string, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	inspectInfo, err := apiClient.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})

	if err != nil {
		return nil, err
	}

	portMap := map[string]bool{}

	if inspectInfo.Container.HostConfig != nil && inspectInfo.Container.HostConfig.PortBindings != nil {
		for _, portBindings := range inspectInfo.Container.HostConfig.PortBindings {
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return "", "", err
	}

	if uid == "" {
		uid = "0"
	}
	execCreate, err := apiClient.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         uid,
	})
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	execAttach, err := apiClient.ExecAttach(ctx, execCreate.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", "", err
	}
	defer execAttach.Close()

	_, err = stdcopy.StdCopy(&stdout, &stderr, execAttach.Reader)
	if err != nil {
		return "", "", err
	}

	info, err := apiClient.ExecInspect(ctx, execCreate.ID, client.ExecInspectOptions{})
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

	ctx, apiClient, err := GetDockerClient()
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

	uid, _, _ := GetContainerUser()
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

	_, err = apiClient.CopyToContainer(ctx, cid.ID, client.CopyToContainerOptions{DestinationPath: dstPath, Content: t, AllowOverwriteDirWithFile: true})
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

	ctx, apiClient, err := GetDockerClient()
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

	reader, err := apiClient.CopyFromContainer(ctx, cid.ID, client.CopyFromContainerOptions{SourcePath: containerPath})
	if err != nil {
		return err
	}

	defer reader.Content.Close()

	_, err = io.Copy(f, reader.Content)
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
	if !term.IsTerminal(os.Stdin.Fd()) || output.JSONOutput {
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
