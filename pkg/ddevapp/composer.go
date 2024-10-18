package ddevapp

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/mattn/go-isatty"
)

// Composer runs Composer commands in the web container, managing pre- and post- hooks
// returns stdout, stderr, error
func (app *DdevApp) Composer(args []string) (string, string, error) {
	err := app.ProcessHooks("pre-composer")
	if err != nil {
		return "", "", fmt.Errorf("failed to process pre-composer hooks: %v", err)
	}

	// Wrap arguments in double quotes if they contain
	// double quotes or single quotes or whitespaces or curly braces.
	// This is needed because we use Cmd (not RawCmd) in ExecOpts
	// to run Composer from $PATH i.e. from `vendor/bin`.
	for i, arg := range args {
		if strings.ContainsAny(arg, `"' {}`) {
			args[i] = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
		}
	}

	argString := strings.Join(args, " ")
	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "web",
		Dir:     app.GetComposerRoot(true, true),
		Cmd:     "composer " + argString,
		Tty:     isatty.IsTerminal(os.Stdin.Fd()),
		// Prevent Composer from debugging when Xdebug is enabled
		Env: []string{"XDEBUG_MODE=off"},
	})
	if err != nil {
		return stdout, stderr, fmt.Errorf("composer command failed: %v", err)
	}

	err = app.MutagenSyncFlush()
	if err != nil {
		return stdout, stderr, err
	}
	if runtime.GOOS == "windows" {
		fileutil.ReplaceSimulatedLinks(app.AppRoot)
	}
	err = app.ProcessHooks("post-composer")
	if err != nil {
		return "", "", fmt.Errorf("failed to process post-composer hooks: %v", err)
	}

	return stdout, stderr, nil
}
