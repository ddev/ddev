// +build !windows

package exec

import (
	"os"
	"strings"

	"github.com/drud/ddev/pkg/util"
)

// RunCommandWithRootRights runs a command with sudo on Unix like systems or
// with elevated rights on Windows.It returns
//stdout of command and an error if any
func RunCommandWithRootRights(command string, args []string) (string, error) {
	if HasRootRights() {
		return RunCommand(command, args)
	}

	newArgs := append([]string{command}, args...)

	util.Success("Running '%s'", strings.Join(newArgs, " "))

	return RunCommand("sudo", newArgs)
}

// HasRootRights returns true if the current context is root on Unix like
// systems or elevated on Windows.
func HasRootRights() bool {
	return os.Geteuid() == 0
}
