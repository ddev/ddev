package globalconfig

import (
	"os"

	"github.com/ddev/ddev/pkg/nodeps"
)

// Container types used with ddev (duplicated from ddevapp, avoiding cross-package cycles)
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
var DdevNoInstrumentation = os.Getenv("DDEV_NO_INSTRUMENTATION") == "true"

// DdevDebug is set to true if the env var is set
var DdevDebug = (os.Getenv("DDEV_DEBUG") == "true")

// DdevVerbose is set to true if the env var is set
var DdevVerbose = (os.Getenv("DDEV_VERBOSE") == "true")

var ValidXdebugIDELocations = []string{XdebugIDELocationContainer, XdebugIDELocationWSL2, ""}

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
