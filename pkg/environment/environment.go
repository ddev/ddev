package environment

import (
	"runtime"

	"github.com/ddev/ddev/pkg/nodeps"
)

const (
	DDEVEnvironmentDarwin          = "darwin"
	DDEVEnvironmentWindows         = "windows"
	DDEVEnvironmentLinux           = "linux"
	DDEVEnvironmentWSL2            = "wsl2"
	DDEVEnvironmentWSL2Mirrored    = "wsl2-mirrored"
	DDEVEnvironmentWSL2VirtioProxy = "wsl2-virtioproxy"
	DDEVEnvironmentWSL2None        = "wsl2-none"
	DDEVEnvironmentCodespaces      = "codespaces"
	DDEVEnvironmentDevcontainer    = "devcontainer"
)

// GetDDEVEnvironment returns the type of environment DDEV is being used in
func GetDDEVEnvironment() string {
	e := runtime.GOOS
	switch {
	case nodeps.IsCodespaces():
		e = DDEVEnvironmentCodespaces
	case nodeps.IsDevcontainer():
		e = DDEVEnvironmentDevcontainer
	case nodeps.IsWSL2MirroredMode():
		e = DDEVEnvironmentWSL2Mirrored
	case nodeps.IsWSL2VirtioProxyMode():
		e = DDEVEnvironmentWSL2VirtioProxy
	case nodeps.IsWSL2NoneMode():
		e = DDEVEnvironmentWSL2None
	case nodeps.IsWSL2():
		e = DDEVEnvironmentWSL2
	}

	return e
}

// IsWSL2Environment returns true for any DDEV environment string that represents a WSL2 mode.
func IsWSL2Environment(envType string) bool {
	switch envType {
	case DDEVEnvironmentWSL2, DDEVEnvironmentWSL2Mirrored, DDEVEnvironmentWSL2VirtioProxy, DDEVEnvironmentWSL2None:
		return true
	default:
		return false
	}
}
