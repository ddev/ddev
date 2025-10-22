package globalconfig

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/moby/term"
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
var DdevNoInstrumentation = os.Getenv("DDEV_NO_INSTRUMENTATION") == "true" || os.Getenv("CI") == "true"

// DdevDebug is set to true if the env var is set
// If DdevVerbose is true, DdevDebug is true
var DdevDebug = os.Getenv("DDEV_DEBUG") == "true" || DdevVerbose

// DdevVerbose is set to true if the env var is set
var DdevVerbose = os.Getenv("DDEV_VERBOSE") == "true"

var ValidXdebugIDELocations = []string{XdebugIDELocationContainer, XdebugIDELocationWSL2, ""}

// GoroutineCount for tests
var GoroutineCount = 0

// IsInteractive returns true if we are running in an interactive mode
func IsInteractive() bool {
	if os.Getenv("DDEV_NONINTERACTIVE") == "true" || os.Getenv("CI") == "true" {
		return false
	}
	// Pretend that terminal is interactive when running tests, because tests may mock input
	if testing.Testing() {
		return true
	}
	if term.IsTerminal(os.Stdin.Fd()) {
		return true
	}
	return false
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
