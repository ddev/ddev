package version

import (
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/version_constants"
	"github.com/fsouza/go-dockerclient"
	"os/exec"
	"runtime"
	"strings"
)

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	var err error
	versionInfo := make(map[string]string)

	versionInfo["DDEV version"] = version_constants.DdevVersion
	versionInfo["web"] = version_constants.GetWebImage()
	versionInfo["db"] = version_constants.GetDBImage(nodeps.MariaDB)
	versionInfo["dba"] = version_constants.GetDBAImage()
	versionInfo["router"] = version_constants.RouterImage + ":" + version_constants.RouterTag
	versionInfo["ddev-ssh-agent"] = version_constants.SSHAuthImage + ":" + version_constants.SSHAuthTag
	versionInfo["build info"] = version_constants.BUILDINFO
	versionInfo["os"] = runtime.GOOS
	versionInfo["architecture"] = runtime.GOARCH
	if versionInfo["docker"], err = dockerutil.GetDockerVersion(); err != nil {
		versionInfo["docker"] = fmt.Sprintf("failed to GetDockerVersion(): %v", err)
	}
	if versionInfo["docker-platform"], err = GetDockerPlatform(); err != nil {
		versionInfo["docker-platform"] = fmt.Sprintf("failed to GetDockerPlatform(): %v", err)
	}
	if versionInfo["docker-compose"], err = dockerutil.GetDockerComposeVersion(); err != nil {
		versionInfo["docker-compose"] = fmt.Sprintf("failed to GetDockerComposeVersion(): %v", err)
	}
	versionInfo["mutagen"] = version_constants.RequiredMutagenVersion

	if runtime.GOOS == "windows" {
		versionInfo["docker type"] = "Docker Desktop For Windows"
	}

	return versionInfo
}

// GetDockerPlatform gets the platform used for docker engine
func GetDockerPlatform() (string, error) {
	var client *docker.Client
	var err error
	if client, err = docker.NewClientFromEnv(); err != nil {
		return "", err
	}

	info, err := client.Info()
	if err != nil {
		return "", err
	}
	platform := info.Name

	return platform, nil
}

// GetLiveMutagenVersion runs `mutagen version` and caches result
func GetLiveMutagenVersion() (string, error) {
	if version_constants.MutagenVersion != "" {
		return version_constants.MutagenVersion, nil
	}

	mutagenPath := globalconfig.GetMutagenPath()

	if !fileutil.FileExists(mutagenPath) {
		version_constants.MutagenVersion = ""
		return version_constants.MutagenVersion, nil
	}
	out, err := exec.Command(mutagenPath, "version").Output()
	if err != nil {
		return "", err
	}

	v := string(out)
	version_constants.MutagenVersion = strings.TrimSpace(v)
	return version_constants.MutagenVersion, nil
}
