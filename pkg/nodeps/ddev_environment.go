package nodeps

import (
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
	case IsCodespaces():
		e = DDEVEnvironmentCodespaces
	case IsGitpod():
		e = DDEVEnvironmentGitpod
	case IsWSL2():
		e = DDEVEnvironmentWSL2
	}
	return e
}
