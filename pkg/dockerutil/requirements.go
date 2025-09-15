package dockerutil

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/versions"
)

type DockerVersionMatrix struct {
	APIVersion               string
	Version                  string
	ComposeVersionConstraint string
}

// DockerRequirements defines the minimum Docker version required by DDEV.
// We compare using the APIVersion because it's a consistent and reliable value.
// The Version is displayed to users as it's more readable and user-friendly.
// The values correspond to the API version matrix found here:
// https://docs.docker.com/reference/api/engine/#api-version-matrix
// List of supported Docker versions: https://endoflife.date/docker-engine
//
// ComposeVersionConstraint is in sync with https://docs.docker.com/desktop/release-notes/
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/ddev/ddev/pull/738 and regression https://github.com/ddev/ddev/issues/1431
var DockerRequirements = DockerVersionMatrix{
	APIVersion:               "1.44",
	Version:                  "25.0",
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
