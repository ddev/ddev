// +build !windows

package exec

import (
	"fmt"
	"strings"
)

// RunBashScriptInteractive runs a bash script on the host system interactively
// with stdin/stdout/stderr connected
func RunBashScriptInteractive(script string, args []string) error {
	err := RunInteractiveCommand(script, args)
	if err != nil {
		return fmt.Errorf("Failed to run \"%s %v\" with error: %v", script, strings.Join(args, " "), err)
	}

	return nil
}
