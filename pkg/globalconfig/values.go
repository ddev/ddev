package globalconfig

import (
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/settings"
)

// Container types used with DDEV (duplicated from ddevapp, avoiding cross-package cycles)
const (
	DdevSSHAgentContainer      = "ddev-ssh-agent"
	DdevRouterContainer        = "ddev-router"
	XdebugIDELocationContainer = "container"
	XdebugIDELocationWSL2      = "wsl2"
)

const DdevGithubOrg = "ddev"

// ValidOmitContainers is the valid omit's that can be done in for a project
var ValidOmitContainers = map[string]bool{
	DdevRouterContainer:   true,
	DdevSSHAgentContainer: true,
}

// DdevNoInstrumentation is set to true if the env var is set
var DdevNoInstrumentation bool

// DdevDebug is set to true if the env var is set
// If DdevVerbose is true, DdevDebug is true
var DdevDebug bool

// DdevVerbose is set to true if the env var is set
var DdevVerbose bool

// RefreshGlobalValues updates the global variables from the settings provider.
// This should be called after settings.Init().
func RefreshGlobalValues() {
	DdevNoInstrumentation = settings.GetBool("NO_INSTRUMENTATION") || settings.GetBool("CI")
	DdevVerbose = settings.GetBool("VERBOSE")
	DdevDebug = settings.GetBool("DEBUG") || DdevVerbose
}

func init() {
	// Initialize with defaults or fallback to Getenv if settings not yet ready
	DdevNoInstrumentation = settings.GetBool("NO_INSTRUMENTATION") || settings.GetBool("CI")
	DdevVerbose = settings.GetBool("VERBOSE")
	DdevDebug = settings.GetBool("DEBUG") || DdevVerbose
}

var ValidXdebugIDELocations = []string{XdebugIDELocationContainer, XdebugIDELocationWSL2, ""}

// GoroutineCount for tests
var GoroutineCount = 0

// IsInteractive returns true if we are running in an interactive mode
func IsInteractive() bool {
	return !settings.GetBool("NONINTERACTIVE")
}

// IsValidXdebugIDELocation limits the choices for XdebugIDELocation
func IsValidXdebugIDELocation(loc string) bool {
	switch {
	case nodeps.ArrayContainsString(ValidXdebugIDELocations, loc):
		return true
	case nodeps.IsIPAddress(loc):
		return true
	}
	return false
}
