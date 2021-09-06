package ddevapp

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
	"strings"
)

// Composer runs composer commands in the web container, managing pre- and post- hooks
// returns stdout, stderr, error
func (app *DdevApp) Composer(args []string) (string, string, error) {
	err := app.ProcessHooks("pre-composer")
	if err != nil {
		return "", "", fmt.Errorf("Failed to process pre-composer hooks: %v", err)
	}

	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "web",
		Dir:     "/var/www/html",
		Cmd:     fmt.Sprintf("composer %s", strings.Join(args, " ")),
		Tty:     isatty.IsTerminal(os.Stdin.Fd()),
	})
	if err != nil {
		return stdout, stderr, fmt.Errorf("composer command failed: %v", err)
	}

	err = app.MutagenSyncFlush()
	if err != nil {
		return stdout, stderr, err
	}
	err = app.ProcessHooks("post-composer")
	if err != nil {
		return "", "", fmt.Errorf("Failed to process post-composer hooks: %v", err)
	}
	return stdout, stderr, nil
}
