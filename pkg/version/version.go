package version

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/environment"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() (map[string]string, error) {
	var retErr error
	trackErr := func(err error) {
		if retErr == nil {
			retErr = err
		}
	}
	versionInfo := make(map[string]string)

	versionInfo["DDEV version"] = versionconstants.DdevVersion
	versionInfo["ddev-environment"] = environment.GetDDEVEnvironment()
	versionInfo["cgo_enabled"] = strconv.FormatInt(versionconstants.CGOEnabled, 10)
	versionInfo["global-ddev-dir"] = util.WindowsPathToCygwinPath(globalconfig.GetGlobalDdevDir())
	versionInfo["go-version"] = runtime.Version()
	versionInfo["web"] = docker.GetWebImage()
	versionInfo["db"] = docker.GetDBImage(nodeps.MariaDB, "")
	versionInfo["router"] = docker.GetRouterImage()
	versionInfo["ddev-ssh-agent"] = docker.GetSSHAuthImage()
	versionInfo["build info"] = versionconstants.BUILDINFO
	versionInfo["os"] = runtime.GOOS
	versionInfo["architecture"] = runtime.GOARCH

	if v, err := dockerutil.GetDockerVersion(); err != nil {
		versionInfo["docker"] = "error"
		trackErr(err)
	} else {
		versionInfo["docker"] = v
	}
	if v, err := dockerutil.GetDockerAPIVersion(); err != nil {
		versionInfo["docker-api"] = "error"
		trackErr(err)
	} else {
		versionInfo["docker-api"] = v
	}
	if v, err := GetDockerPlatform(); err != nil {
		versionInfo["docker-platform"] = "error"
		trackErr(err)
	} else {
		versionInfo["docker-platform"] = v
	}
	if v, err := dockerutil.GetDockerBuildxVersion(); err != nil {
		versionInfo["docker-buildx"] = "error"
	} else {
		versionInfo["docker-buildx"] = v
	}
	versionInfo["mutagen"] = versionconstants.RequiredMutagenVersion
	versionInfo["xhgui-image"] = docker.GetXhguiImage()

	return versionInfo, retErr
}

// GetDockerPlatform gets the platform used for Docker engine
func GetDockerPlatform() (string, error) {
	info, err := dockerutil.GetDockerClientInfo()
	if err != nil {
		return "", err
	}

	platform := info.OperatingSystem
	switch {
	case dockerutil.IsDockerDesktop():
		platform = "docker-desktop"
	case dockerutil.IsRancherDesktop():
		platform = "rancher-desktop"
	case dockerutil.IsColima():
		platform = "colima"
	case dockerutil.IsLima():
		platform = "lima"
	case dockerutil.IsOrbStack():
		platform = "orbstack"
	case dockerutil.IsPodman():
		platform = "podman"
	case nodeps.IsWSL2() && info.OSType == "linux":
		platform = "wsl2-docker-ce"
	case !nodeps.IsWSL2() && info.OSType == "linux":
		platform = "linux-docker"
	}

	if dockerutil.IsRootless() {
		platform += "-rootless"
	}

	if dockerutil.IsSELinux() {
		platform += "-selinux"
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
