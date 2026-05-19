package dockerutil

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/client/pkg/versions"
)

type DockerVersionMatrix struct {
	APIVersion    string
	Version       string
	PodmanVersion string
}

// DockerRequirements defines the minimum Docker version recommended by DDEV.
// We compare using the APIVersion because it's a consistent and reliable value.
// The Version is displayed to users as it's more readable and user-friendly.
var DockerRequirements = DockerVersionMatrix{
	APIVersion:    versionconstants.DockerMinAPIVersion,
	Version:       versionconstants.DockerMinVersion,
	PodmanVersion: versionconstants.PodmanMinVersion,
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

	if IsPodman() {
		if dockerVersionMatrix.PodmanVersion == "" {
			return fmt.Errorf("the check is done against Docker, but Podman is detected. Podman is not supported at this time")
		}
		if !versions.GreaterThanOrEqualTo(currentVersion, dockerVersionMatrix.PodmanVersion) {
			return fmt.Errorf("installed Podman version %s is not supported, please update to version %s or newer", currentVersion, dockerVersionMatrix.PodmanVersion)
		}
		return nil
	}

	// Check against recommended API version, if it fails, suggest the minimum Docker version that relates to supported API
	if !versions.GreaterThanOrEqualTo(currentAPIVersion, dockerVersionMatrix.APIVersion) {
		return fmt.Errorf("installed Docker version %s is not supported, please update to version %s or newer", currentVersion, dockerVersionMatrix.Version)
	}
	return nil
}

// CanRunWithoutDocker returns true if the command or flag can run without Docker.
func CanRunWithoutDocker() bool {
	if len(os.Args) < 2 {
		return true
	}
	// Some commands don't support Cobra help, because they are wrappers
	if slices.Contains([]string{"composer"}, os.Args[1]) {
		return false
	}
	if output.ParseBoolFlag("version", "v") || output.ParseBoolFlag("help", "h") {
		return true
	}
	if len(os.Args) == 2 && output.ParseBoolFlag("json-output", "j") {
		return true
	}
	// Some commands don't require docker
	if slices.Contains([]string{"config", "help", "hostname", "version"}, os.Args[1]) {
		return true
	}
	return false
}

// CheckAvailableSpace returns an error if Docker space is low
func CheckAvailableSpace() error {
	// stat -f -c "%a %b %S" / outputs three space-separated values:
	// %a = free blocks available to non-root, %b = total blocks, %S = block size in bytes
	_, out, _ := RunSimpleContainer(versionconstants.UtilitiesImage, "check-available-space-"+util.RandString(6), []string{"stat", "-f", "-c", "%a %b %S", "/"}, []string{}, []string{}, []string{}, "", true, false, nil, nil, nil)
	var availBlocks, totalBlocks, blockSize int64
	if n, err := fmt.Sscanf(strings.TrimSpace(out), "%d %d %d", &availBlocks, &totalBlocks, &blockSize); n != 3 || err != nil {
		return fmt.Errorf("unable to determine available disk space from %q: %v", out, err)
	}
	spaceAvail := availBlocks * blockSize
	spaceTotal := totalBlocks * blockSize
	if spaceTotal == 0 {
		return fmt.Errorf("unable to determine available disk space: total space is zero")
	}
	if spaceAvail < nodeps.MinimumDockerSpaceWarning {
		spacePercent := int((spaceTotal - spaceAvail) * 100 / spaceTotal)
		return fmt.Errorf("your Docker install has only %s available disk space, less than %s warning level (%d%% used). Please increase disk image size. More info at %s", util.FormatBytes(spaceAvail), util.FormatBytes(nodeps.MinimumDockerSpaceWarning), spacePercent, "https://docs.ddev.com/en/stable/users/usage/troubleshooting/#out-of-disk-space")
	}
	return nil
}

// CheckDockerAuth checks if Docker authentication is properly configured
func CheckDockerAuth() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, ".docker", "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file is not necessarily an error - Docker can work without it
			return nil
		}
		return fmt.Errorf("unable to read Docker config: %v", err)
	}

	var config struct {
		CredsStore  string            `json:"credsStore"`
		CredHelpers map[string]string `json:"credHelpers"`
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("unable to parse Docker config: %v", err)
	}

	// If credsStore is set, verify the credential helper exists
	if config.CredsStore != "" {
		helperName := "docker-credential-" + config.CredsStore
		_, err := osexec.LookPath(helperName)
		if err != nil {
			return fmt.Errorf("credsStore is set to '%s' in ~/.docker/config.json but '%s' is not found in PATH. This will cause 'ddev start' to fail when pulling images. Either install the credential helper or remove the 'credsStore' line from ~/.docker/config.json. See https://docs.docker.com/reference/cli/docker/login/#credential-helpers", config.CredsStore, helperName)
		}
	}

	return nil
}
