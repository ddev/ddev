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
	DDEVEnvironmentWSL2Bridged     = "wsl2-bridged"
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
	case nodeps.IsWSL2():
		e = wsl2Environment()
	}

	return e
}

// wsl2Environment returns the specific WSL2 environment type by calling
// GetWSL2NetworkingMode once. nat is the common case; others are unusual.
func wsl2Environment() string {
	mode, err := nodeps.GetWSL2NetworkingMode()
	if err != nil {
		return DDEVEnvironmentWSL2
	}
	switch mode {
	case "mirrored":
		return DDEVEnvironmentWSL2Mirrored
	case "virtioproxy":
		return DDEVEnvironmentWSL2VirtioProxy
	case "none":
		return DDEVEnvironmentWSL2None
	case "bridged":
		return DDEVEnvironmentWSL2Bridged
	default: // "nat" is the normal case
		return DDEVEnvironmentWSL2
	}
}

// IsWSL2Environment returns true for any DDEV environment string that represents a WSL2 mode.
func IsWSL2Environment(envType string) bool {
	switch envType {
	case DDEVEnvironmentWSL2, DDEVEnvironmentWSL2Mirrored, DDEVEnvironmentWSL2VirtioProxy, DDEVEnvironmentWSL2None, DDEVEnvironmentWSL2Bridged:
		return true
	default:
		return false
	}
}
