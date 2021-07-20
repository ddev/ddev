package globalconfig

import "os"

// Container types used with ddev (duplicated from ddevapp, avoiding cross-package cycles)
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBAContainer          = "dba"
)

// ValidOmitContainers is the valid omit's that can be done in for a project
var ValidOmitContainers = map[string]bool{
	DdevSSHAgentContainer: true,
	DBAContainer:          true,
}

// DdevNoInstrumentation is set to true if the env var is set
var DdevNoInstrumentation = os.Getenv("DDEV_NO_INSTRUMENTATION") == "true"

// DdevDebug is set to true if the env var is set
var DdevDebug = (os.Getenv("DDEV_DEBUG") == "true")

// DdevVerbose is set to true if the env var is set
var DdevVerbose = (os.Getenv("DDEV_VERBOSE") == "true")
