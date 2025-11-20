package ddevapp

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/mattn/go-isatty"
)

// Composer runs Composer commands in the web container, managing pre- and post- hooks
// returns stdout, stderr, error
func (app *DdevApp) Composer(args []string) (string, string, error) {
	err := app.ProcessHooks("pre-composer")
	if err != nil {
		return "", "", fmt.Errorf("failed to process pre-composer hooks: %v", err)
	}

	stdout, stderr, err := app.Exec(&ExecOpts{
		Service: "web",
		Dir:     app.GetComposerRoot(true, true),
		RawCmd:  append([]string{"composer"}, args...),
		Tty:     isatty.IsTerminal(os.Stdin.Fd()),
		Env:     getComposerEnv(),
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
	err = app.ProcessHooks("post-composer")
	if err != nil {
		return "", "", fmt.Errorf("failed to process post-composer hooks: %v", err)
	}

	return stdout, stderr, nil
}

// getComposerEnv returns environment variables to use when running composer
func getComposerEnv() []string {
	env := []string{
		// Prevent Composer from debugging when Xdebug is enabled
		"XDEBUG_MODE=off",
	}

	// List of Composer environment variables to pass through from host
	// https://getcomposer.org/doc/03-cli.md#environment-variables
	composerEnvVars := []string{
		"COMPOSER_NO_SECURITY_BLOCKING",
	}

	for _, varName := range composerEnvVars {
		if value, exists := os.LookupEnv(varName); exists {
			env = append(env, fmt.Sprintf(`%s=%s`, varName, value))
		}
	}

	return env
}
