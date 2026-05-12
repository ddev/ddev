package dockerutil

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/client/pkg/versions"
)

// DownloadDockerBuildxIfNeeded downloads the proper version of docker-buildx
// if it's either not yet installed or doesn't meet the minimum version requirement.
// Returns downloaded bool (true if it did the download) and err.
func DownloadDockerBuildxIfNeeded() (bool, error) {
	// Don't download anything if there are problems with Docker
	_, err := getDockerManagerInstance()
	if err != nil {
		return false, err
	}

	dockerBuildxResetMsg := fmt.Sprintf(`
To use the system docker-buildx (no download, recommended):
  ddev config global --docker-buildx-version=system

To download a specific docker-buildx (only if you can't use the system buildx):
  ddev config global --docker-buildx-version=%s

More details in https://ddev.com/blog/docker-buildx-requirement-v1-25-1/`, versionconstants.DockerBuildxRecommendedVersion)

	requiredVersion := globalconfig.GetRequiredDockerBuildxVersion()
	err = CheckDockerBuildxVersion()
	if err != nil {
		err = fmt.Errorf("%w\n%s", err, dockerBuildxResetMsg)
	}
	// If no required version is set, then we don't need to download
	// but if there was an error checking the version, report it
	if requiredVersion == "" {
		return false, err
	}
	currentVersion, _ := GetDockerBuildxVersion()
	// Skip the download if the version matches and DDEV's own binary already exists.
	destinationPath, _ := globalconfig.GetDockerBuildxDestination()
	_, destStatErr := os.Stat(destinationPath)
	if currentVersion == requiredVersion && destStatErr == nil {
		return false, err
	}
	// If we get here, we need to download the required version.
	// If that fails, report the error but also include any error
	// from the version check (e.g., if docker-buildx isn't found at all)
	downloadErr := DownloadDockerBuildx()
	if downloadErr == nil {
		// Reset the plugins to pick up the new binary
		_ = ResetCLIPlugins()
		if err = CheckDockerBuildxVersion(); err != nil {
			return false, fmt.Errorf("%w\n%s", err, dockerBuildxResetMsg)
		}
		return true, nil
	}
	return false, fmt.Errorf("unable to download required docker-buildx version %q: %w\n%s", requiredVersion, downloadErr, dockerBuildxResetMsg)
}

// DownloadDockerBuildx gets the docker-buildx binary and puts it into
// ~/.ddev/bin
func DownloadDockerBuildx() error {
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile, _ := globalconfig.GetDockerBuildxDestination()

	buildxURL, shasumURL, err := dockerBuildxDownloadLink()
	if err != nil {
		return err
	}
	util.Debug("Downloading '%s' to '%s' ...", buildxURL, destFile)

	_ = os.Remove(destFile)

	_ = os.MkdirAll(globalBinDir, 0777)
	err = util.DownloadFile(destFile, buildxURL, globalconfig.IsInteractive(), shasumURL)
	if err != nil {
		_ = os.Remove(destFile)
		return err
	}
	output.UserErr.Printf("Download complete.")

	err = util.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}

// GetDockerBuildxVersion returns the version of the Docker buildx plugin.
func GetDockerBuildxVersion() (string, error) {
	plugin, err := GetCLIPlugin("buildx")
	if err != nil {
		return "", err
	}
	// Strip leading "v" prefix if present for semver compatibility
	return strings.TrimPrefix(plugin.Version, "v"), nil
}

// GetDockerBuildxLocation returns the path to the Docker buildx plugin.
func GetDockerBuildxLocation() (string, error) {
	plugin, err := GetCLIPlugin("buildx")
	if err != nil {
		return "", err
	}
	return plugin.Path, nil
}

// CheckDockerBuildxVersion checks that the Docker buildx plugin is installed
// and meets the minimum version requirement.
func CheckDockerBuildxVersion() error {
	defer util.TimeTrack()()

	v, err := GetDockerBuildxVersion()
	if err != nil {
		return fmt.Errorf("compose build requires buildx %s or later: %v", versionconstants.DockerBuildxMinVersion, err)
	}

	pluginPath, _ := GetDockerBuildxLocation()

	if versions.LessThan(v, versionconstants.DockerBuildxMinVersion) {
		return fmt.Errorf("compose build requires buildx %s or later.\nInstalled docker-buildx: %s (plugin path: %q)", versionconstants.DockerBuildxMinVersion, v, pluginPath)
	}

	return nil
}

// dockerBuildxDownloadLink returns the URL and SHASUM-file link for docker-buildx
func dockerBuildxDownloadLink() (buildxURL string, shasumURL string, err error) {
	arch := runtime.GOARCH

	if arch != "arm64" && arch != "amd64" {
		return "", "", fmt.Errorf("only ARM64 and AMD64 architectures are supported for docker-buildx, not %s", arch)
	}
	flavor := runtime.GOOS + "-" + arch
	version := globalconfig.GetRequiredDockerBuildxVersion()
	if version == "" {
		return "", "", fmt.Errorf("docker-buildx version is not set in global config")
	}
	binaryURL := fmt.Sprintf("https://github.com/docker/buildx/releases/download/v%[1]s/buildx-v%[1]s.%[2]s", version, flavor)
	if nodeps.IsWindows() {
		binaryURL = binaryURL + ".exe"
	}
	shasumURL = fmt.Sprintf("https://github.com/docker/buildx/releases/download/v%s/checksums.txt", version)
	// darwin binaries are not included in the buildx checksums file, skip sha verification
	// see https://github.com/docker/buildx/issues/2727
	if nodeps.IsMacOS() {
		shasumURL = ""
	}

	return binaryURL, shasumURL, nil
}
