package dockerutil

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/client/pkg/versions"
)

type DockerVersionMatrix struct {
	APIVersion               string
	Version                  string
	BuildxVersion            string
	PodmanVersion            string
	ComposeVersionConstraint string
}

// DockerRequirements defines the minimum Docker version required by DDEV.
// We compare using the APIVersion because it's a consistent and reliable value.
// The Version is displayed to users as it's more readable and user-friendly.
// The values correspond to the API version matrix found here:
// https://docs.docker.com/reference/api/engine/#api-version-matrix
// List of supported Docker versions: https://endoflife.date/docker-engine
//
// PodmanVersion is the minimum Podman version supported. If it's empty, Podman is not supported.
// Podman 4.x doesn't run properly on GitHub runners, so we require Podman 5.0+
//
// ComposeVersionConstraint is in sync with https://docs.docker.com/desktop/release-notes/
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/ddev/ddev/pull/738 and regression https://github.com/ddev/ddev/issues/1431
var DockerRequirements = DockerVersionMatrix{
	APIVersion:               "1.44",
	Version:                  "25.0",
	BuildxVersion:            "0.17.0",
	PodmanVersion:            "5.0",
	ComposeVersionConstraint: ">= 2.24.3",
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

// CheckDockerCompose determines if docker-compose is present and executable on the host system. This
// relies on docker-compose being somewhere in the user's $PATH.
func CheckDockerCompose() error {
	defer util.TimeTrack()()

	_, err := DownloadDockerComposeIfNeeded()
	if err != nil {
		return err
	}
	versionConstraint := DockerRequirements.ComposeVersionConstraint

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
	// help and hostname should not require docker
	if slices.Contains([]string{"help", "hostname"}, os.Args[1]) {
		return true
	}
	return false
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
		util.Error("Your Docker install has only %d available disk space, less than %d warning level (%d%% used). Please increase disk image size. More info at %s", spaceAbsolute, nodeps.MinimumDockerSpaceWarning, spacePercent, "https://docs.ddev.com/en/stable/users/usage/troubleshooting/#out-of-disk-space")
	}
}

// GetBuildxVersion returns the version of the Docker buildx plugin.
func GetBuildxVersion() (string, error) {
	plugin, err := GetCLIPlugin("buildx")
	if err != nil {
		return "", err
	}
	// Strip leading "v" prefix if present for semver compatibility
	return strings.TrimPrefix(plugin.Version, "v"), nil
}

// GetBuildxLocation returns the path to the Docker buildx plugin.
func GetBuildxLocation() (string, error) {
	plugin, err := GetCLIPlugin("buildx")
	if err != nil {
		return "", err
	}
	return plugin.Path, nil
}

// CheckDockerBuildxVersion checks that the Docker buildx plugin is installed
// and meets the minimum version requirement.
func CheckDockerBuildxVersion(dockerVersionMatrix DockerVersionMatrix) error {
	defer util.TimeTrack()()

	v, err := GetBuildxVersion()
	if err != nil {
		return fmt.Errorf("compose build requires buildx %s or later: %v.\nPlease install buildx: https://github.com/docker/buildx#installing", dockerVersionMatrix.BuildxVersion, err)
	}

	pluginPath, err := GetBuildxLocation()
	if err != nil {
		pluginPath = "unknown"
	}

	if versions.LessThan(v, dockerVersionMatrix.BuildxVersion) {
		return fmt.Errorf("compose build requires buildx %s or later.\nInstalled docker buildx: %s (plugin path: %s)\nPlease update buildx: https://github.com/docker/buildx#installing", dockerVersionMatrix.BuildxVersion, v, pluginPath)
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
