package version

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerClient "github.com/docker/docker/client"
)

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	var err error
	versionInfo := make(map[string]string)

	versionInfo["DDEV version"] = versionconstants.DdevVersion
	versionInfo["cgo_enabled"] = strconv.FormatInt(versionconstants.CGOEnabled, 10)
	versionInfo["global-ddev-dir"] = globalconfig.GetGlobalDdevDir()
	versionInfo["web"] = docker.GetWebImage()
	versionInfo["db"] = docker.GetDBImage(nodeps.MariaDB, "")
	versionInfo["router"] = docker.GetRouterImage()
	versionInfo["ddev-ssh-agent"] = docker.GetSSHAuthImage()
	versionInfo["build info"] = versionconstants.BUILDINFO
	versionInfo["os"] = runtime.GOOS
	versionInfo["architecture"] = runtime.GOARCH
	if versionInfo["docker"], err = dockerutil.GetDockerVersion(); err != nil {
		versionInfo["docker"] = fmt.Sprintf("Failed to GetDockerVersion(): %v", err)
	}
	if versionInfo["docker-api"], err = dockerutil.GetDockerAPIVersion(); err != nil {
		versionInfo["docker-api"] = fmt.Sprintf("Failed to GetDockerAPIVersion(): %v", err)
	}
	if versionInfo["docker-platform"], err = GetDockerPlatform(); err != nil {
		versionInfo["docker-platform"] = fmt.Sprintf("Failed to GetDockerPlatform(): %v", err)
	}
	if versionInfo["docker-compose"], err = dockerutil.GetDockerComposeVersion(); err != nil {
		versionInfo["docker-compose"] = fmt.Sprintf("Failed to GetDockerComposeVersion(): %v", err)
	}
	versionInfo["mutagen"] = versionconstants.RequiredMutagenVersion

	if runtime.GOOS == "windows" {
		versionInfo["docker type"] = "Docker Desktop For Windows"
	}

	return versionInfo
}

// GetDockerPlatform gets the platform used for Docker engine
func GetDockerPlatform() (string, error) {
	ctx := context.Background()
	var client *dockerClient.Client
	var err error
	if client, err = dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation()); err != nil {
		return "", err
	}

	info, err := client.Info(ctx)
	if err != nil {
		return "", err
	}

	platform := info.OperatingSystem
	switch {
	case strings.HasPrefix(platform, "Docker Desktop"):
		platform = "docker-desktop"
	case strings.HasPrefix(platform, "Rancher Desktop") || strings.Contains(info.Name, "rancher-desktop"):
		platform = "rancher-desktop"
	case strings.HasPrefix(info.Name, "colima"):
		platform = "colima"
	case strings.HasPrefix(info.Name, "lima"):
		platform = "lima"
	case platform == "OrbStack":
		platform = "orbstack"
	case nodeps.IsWSL2() && info.OSType == "linux":
		platform = "wsl2-docker-ce"
	case !nodeps.IsWSL2() && info.OSType == "linux":
		platform = "linux-docker"
	default:
		platform = info.OperatingSystem
	}

	return platform, nil
}

// GetLiveMutagenVersion runs `mutagen version` and caches result
func GetLiveMutagenVersion() (string, error) {
	if versionconstants.MutagenVersion != "" {
		return versionconstants.MutagenVersion, nil
	}

	mutagenPath := globalconfig.GetMutagenPath()

	if !fileutil.FileExists(mutagenPath) {
		versionconstants.MutagenVersion = ""
		return versionconstants.MutagenVersion, nil
	}
	out, err := exec.Command(mutagenPath, "version").Output()
	if err != nil {
		return "", err
	}

	v := string(out)
	versionconstants.MutagenVersion = strings.TrimSpace(v)
	return versionconstants.MutagenVersion, nil
}
