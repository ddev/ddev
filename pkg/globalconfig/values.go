package globalconfig

import "os"

// Container types used with ddev (duplicated from ddevapp, avoiding cross-package cycles)
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBAContainer          = "dba"
)

var ValidOmitContainers = map[string]bool{
	DdevSSHAgentContainer: true,
	DBAContainer:          true,
}

var DdevNoInstrumentation = os.Getenv("DDEV_NO_INSTRUMENTATION") == "true"
var DdevDebug = (os.Getenv("DDEV_DEBUG") == "true")
var DdevVerbose = (os.Getenv("DDEV_VERBOSE") == "true")
