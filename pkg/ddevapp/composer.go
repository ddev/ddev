package ddevapp

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
)

// Composer runs Composer commands in the web container, managing pre- and post- hooks
// returns stdout, stderr, error
func (app *DdevApp) Composer(args []string) (string, string, error) {
	return app.composer(args, false)
}

// composer runs Composer, skipping the composer and exec hooks when skipHooks
// is set, so a composer task inside one of those hooks cannot run itself.
func (app *DdevApp) composer(args []string, skipHooks bool) (string, string, error) {
	if !skipHooks {
		if err := app.ProcessHooks("pre-composer"); err != nil {
			return "", "", fmt.Errorf("failed to process pre-composer hooks: %v", err)
		}
	}

	stdout, stderr, err := app.Exec(&ExecOpts{
		Service:   "web",
		Dir:       app.GetComposerRoot(true, true),
		RawCmd:    append([]string{"composer"}, args...),
		Tty:       isatty.IsTerminal(os.Stdin.Fd()),
		Env:       getComposerEnv(),
		SkipHooks: skipHooks,
	})
	if err != nil {
		return stdout, stderr, fmt.Errorf("composer command failed: %v", err)
	}

	err = app.MutagenSyncFlush()
	if err != nil {
		return stdout, stderr, err
	}
	if nodeps.IsWindows() {
		fileutil.ReplaceSimulatedLinks(app.AppRoot)
	}
	if !skipHooks {
		if err := app.ProcessHooks("post-composer"); err != nil {
			return "", "", fmt.Errorf("failed to process post-composer hooks: %v", err)
		}
	}

	return stdout, stderr, nil
}

// composerEnvVars are the Composer variables passed through from the host.
// COMPOSER_NO_SECURITY_BLOCKING is deprecated in favor of COMPOSER_NO_BLOCKING,
// but still forwarded for anyone who already sets it.
// See https://getcomposer.org/doc/03-cli.md#environment-variables
var composerEnvVars = []string{
	"COMPOSER_NO_BLOCKING",
	"COMPOSER_NO_SECURITY_BLOCKING",
}

// getComposerEnv returns environment variables to use when running composer
func getComposerEnv() []string {
	// Prevent Composer from debugging when Xdebug is enabled
	return append([]string{"XDEBUG_MODE=off"}, util.HostEnv(composerEnvVars...)...)
}
