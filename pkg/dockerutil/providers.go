package dockerutil

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
)

// IsDockerDesktop detects if running on Docker Desktop
func IsDockerDesktop() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "Docker Desktop") {
		return true
	}
	if strings.Contains(info.Name, "docker-desktop") {
		return true
	}
	return false
}

// IsColima detects if running on Colima
func IsColima() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.Name, "colima") {
		return true
	}
	return false
}

// IsLima detects if running on lima
func IsLima() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	// Rancher Desktop uses "lima-rancher-desktop" as its name
	if strings.Contains(info.Name, "rancher-desktop") {
		return false
	}
	if strings.HasPrefix(info.Name, "lima") {
		return true
	}
	return false
}

// IsRancherDesktop detects if running on Rancher Desktop
func IsRancherDesktop() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "Rancher Desktop") {
		return true
	}
	if strings.Contains(info.Name, "rancher-desktop") {
		return true
	}
	return false
}

// IsOrbStack detects if running on OrbStack
func IsOrbStack() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	if strings.HasPrefix(info.OperatingSystem, "OrbStack") {
		return true
	}
	if strings.Contains(info.Name, "orbstack") {
		return true
	}
	return false
}

// IsPodman detects if running on Podman (either rootless or root)
func IsPodman() bool {
	serverVersion, err := GetServerVersion()
	if err != nil {
		return false
	}
	for _, v := range serverVersion.Components {
		if strings.HasPrefix(v.Name, "Podman Engine") {
			return true
		}
	}
	return false
}

// IsRootless detects if Docker is running in rootless mode
func IsRootless() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	return slices.Contains(info.SecurityOptions, "name=rootless")
}

// IsPodmanRootless detects if Podman is running in rootless mode
func IsPodmanRootless() bool {
	return IsRootless() && IsPodman()
}

func IsPodmanRootlessmacOS() bool {
	return nodeps.IsMacOS() && IsRootless() && IsPodman()
}

// UseKeepID reports whether containers and volume operations should use Podman's
// "keep-id" user-namespace mode. This is the single source of truth for the
// decision: every container we create and every volume copy/chown must agree,
// because they share volumes (notably ddev-global-cache) and a uid written
// under one userns mapping is unreadable under another.
//
// keep-id is correct only on Linux rootless Podman, where it maps the host
// uid/gid into the container so bind-mounted files have the host's ownership.
// On macOS/Windows, Podman runs in a Linux VM whose subuid/subgid range cannot
// map host GIDs (e.g. macOS GID 20/staff), so keep-id must not be used; all
// containers run under Podman's default rootless userns instead.
func UseKeepID() bool {
	return IsPodmanRootless() && nodeps.IsLinux()
}

// IsDockerRootless detects if Docker is running in rootless mode on Linux
// It must not be Podman or Lima, which can be rootless as well.
func IsDockerRootless() bool {
	return IsRootless() && nodeps.IsLinux() && !IsPodman() && !IsLima()
}

// IsSELinux detects if SELinux is enabled
func IsSELinux() bool {
	info, err := GetDockerClientInfo()
	if err != nil {
		return false
	}
	return slices.Contains(info.SecurityOptions, "name=selinux")
}

// IsSocktainer detects if the Docker provider is socktainer, the
// Docker-compatible API that fronts Apple Container.
func IsSocktainer() bool {
	serverVersion, err := GetServerVersion()
	if err != nil {
		return false
	}
	if serverVersion.Platform.Name == "socktainer" {
		return true
	}
	for _, v := range serverVersion.Components {
		if strings.HasPrefix(v.Name, "socktainer") {
			return true
		}
	}
	return false
}

// UseBindGlobalCache is true when /mnt/ddev-global-cache should be a bind mount of a
// host directory instead of the ddev-global-cache Docker volume. It says nothing about
// the project bind mounts controlled by the no_bind_mounts global config setting.
//
// Apple Container backs each named volume with an ext4 block image that only one
// running container can attach read-write, so a volume mounted at the same time by
// web, db and the router cannot work there. Host bind mounts are virtiofs-backed and
// can be shared. This turns on automatically on socktainer; DDEV_BIND_GLOBAL_CACHE
// forces it on or off for testing against other providers.
func UseBindGlobalCache() bool {
	if v := os.Getenv("DDEV_BIND_GLOBAL_CACHE"); v != "" {
		return v == "true"
	}
	return IsSocktainer()
}

// GlobalCacheSource returns what to use as the source of the
// /mnt/ddev-global-cache mount: either the Docker volume name, or a host
// directory path when UseBindGlobalCache() is true.
func GlobalCacheSource() string {
	if UseBindGlobalCache() {
		return filepath.Join(globalconfig.GetGlobalDdevDir(), "global-cache-bind")
	}
	return "ddev-global-cache"
}

// GlobalCacheMount returns the full mount spec for /mnt/ddev-global-cache.
func GlobalCacheMount() string {
	return GlobalCacheSource() + ":/mnt/ddev-global-cache"
}
