package environment

import (
	"runtime"

	"github.com/ddev/ddev/pkg/nodeps"
)

const (
	DDEVEnvironmentDarwin       = "darwin"
	DDEVEnvironmentWindows      = "windows"
	DDEVEnvironmentLinux        = "linux"
	DDEVEnvironmentWSL2         = "wsl2"
	DDEVEnvironmentWSL2Mirrored = "wsl2-mirrored"
	DDEVEnvironmentCodespaces   = "codespaces"
)

// GetDDEVEnvironment returns the type of environment DDEV is being used in
func GetDDEVEnvironment() string {
	e := runtime.GOOS
	switch {
	case nodeps.IsCodespaces():
		e = DDEVEnvironmentCodespaces
	case nodeps.IsWSL2MirroredMode():
		e = DDEVEnvironmentWSL2Mirrored
	case nodeps.IsWSL2():
		e = DDEVEnvironmentWSL2
	}

	return e
}
