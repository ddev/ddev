package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/mattn/go-isatty"
	"os"
	"runtime"
	"strings"
)

// Composer runs Composer commands in the web container, managing pre- and post- hooks
// returns stdout, stderr, error
func (app *DdevApp) Composer(args []string) (string, string, error) {
	// Set composer command context for hooks (backward compatible - only adds env vars)
	if len(args) > 0 {
		os.Setenv("DDEV_COMPOSER_COMMAND", args[0])
		os.Setenv("DDEV_COMPOSER_ARGS", strings.Join(args, " "))
		defer os.Unsetenv("DDEV_COMPOSER_COMMAND")
		defer os.Unsetenv("DDEV_COMPOSER_ARGS")
	}

	err := app.ProcessHooks("pre-composer")
	if err != nil {
		return "", "", fmt.Errorf("failed to process pre-composer hooks: %v", err)
	}

	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "web",
		Dir:     app.GetComposerRoot(true, true),
		RawCmd:  append([]string{"composer"}, args...),
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
