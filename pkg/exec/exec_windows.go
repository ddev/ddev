package exec

import (
	"fmt"
	"strings"

	"github.com/drud/ddev/pkg/util"
)

// RunBashScriptInteractive runs a bash script on the host system interactively
// with stdin/stdout/stderr connected
func RunBashScriptInteractive(script string, args []string) error {
	bashPath, err := util.FindBashPath()
	if err != nil {
		return err
	}

	realArgs := []string{script}
	realArgs = append(realArgs, args...)

	err = RunInteractiveCommand(bashPath, realArgs)
	if err != nil {
		return fmt.Errorf("Failed to run \"%s %v\" with error: %v", bashPath, strings.Join(realArgs, " "), err)
	}

	return nil
}
