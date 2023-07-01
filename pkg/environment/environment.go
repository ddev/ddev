package environment

import (
	"github.com/ddev/ddev/pkg/nodeps"
	"runtime"
)

const (
	DDEVEnvironmentDarwin     = "darwin"
	DDEVEnvironmentWindows    = "windows"
	DDEVEnvironmentLinux      = "linux"
	DDEVEnvironmentWSL2       = "wsl2"
	DDEVEnvironmentGitpod     = "gitpod"
	DDEVEnvironmentCodespaces = "codespaces"
)

// GetDDEVEnvironment returns the type of environment DDEV is being used in
func GetDDEVEnvironment() string {
	e := runtime.GOOS
	switch {
	case nodeps.IsCodespaces():
		e = DDEVEnvironmentCodespaces
	case nodeps.IsGitpod():
		e = DDEVEnvironmentGitpod
	case nodeps.IsWSL2():
		e = DDEVEnvironmentWSL2
	}
	return e
}
