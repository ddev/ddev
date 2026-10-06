package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestExecEnv tests which host variables reach each service, that an env file
// for web keeps its value over a bare name, and that the env of the caller
// wins over all of them.
func TestExecEnv(t *testing.T) {
	// The agent and terminal running the tests must not leak into the cases.
	// t.Setenv registers the restore, and an empty value would be forwarded.
	for _, name := range append(aiAgentEnvVars, util.TerminalEnvVars...) {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	// TerminalEnv replaces a TERM the container cannot resolve, a bare TERM
	// in web_environment does not.
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("AI_AGENT", "test-agent")
	t.Setenv("COPILOT_GITHUB_TOKEN", "secret")
	t.Setenv("DDEV_TEST_GLOBAL_BARE", "global")
	t.Setenv("DDEV_TEST_PROJECT_BARE", "project")
	t.Setenv("DDEV_TEST_OVERRIDDEN", "host")
	t.Setenv("DDEV_TEST_WITH_VALUE", "host")
	t.Setenv("DDEV_TEST_PROJECT_REBARED", "host")
	t.Setenv("DDEV_TEST_BOTH_BARE", "both")
	t.Setenv("DDEV_TEST_IN_WEB_FILE", "host")
	t.Setenv("DDEV_TEST_IN_SHARED_FILE", "host")
	t.Setenv("DDEV_TEST_IN_DB_FILE", "host")

	origGlobal := globalconfig.DdevGlobalConfig.WebEnvironment
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.WebEnvironment = origGlobal })
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"DDEV_TEST_GLOBAL_BARE", "DDEV_TEST_OVERRIDDEN", "DDEV_TEST_PROJECT_REBARED=config", "DDEV_TEST_BOTH_BARE"}
	app := &DdevApp{
		AppRoot:        t.TempDir(),
		WebEnvironment: []string{"DDEV_TEST_PROJECT_BARE", "DDEV_TEST_OVERRIDDEN=config", "DDEV_TEST_WITH_VALUE=config", "DDEV_TEST_UNSET", "DDEV_TEST_PROJECT_REBARED", "DDEV_TEST_BOTH_BARE", "DDEV_TEST_IN_WEB_FILE", "DDEV_TEST_IN_SHARED_FILE", "DDEV_TEST_IN_DB_FILE", "TERM"},
	}
	require.NoError(t, os.MkdirAll(app.AppConfDir(), 0755))
	for file, content := range map[string]string{
		".env.web.label": "DDEV_TEST_IN_WEB_FILE=file\n",
		".env":           "DDEV_TEST_IN_SHARED_FILE=file\n",
		".env.db":        "DDEV_TEST_IN_DB_FILE=file\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(app.AppConfDir(), file), []byte(content), 0644))
	}

	// A bare name is forwarded from global or project config, unless the
	// other one gives it a value and wins: the project over the global.
	// An env file that reaches web keeps the name out, one for db does not.
	webBare := []string{"DDEV_TEST_GLOBAL_BARE=global", "DDEV_TEST_PROJECT_BARE=project", "DDEV_TEST_PROJECT_REBARED=host", "DDEV_TEST_BOTH_BARE=both", "DDEV_TEST_IN_DB_FILE=host"}
	testCases := []struct {
		service   string
		tty       bool
		callerEnv []string
		expected  []string
	}{
		{"web", false, nil, append(webBare, "TERM=xterm-kitty", "AI_AGENT=test-agent")},
		{"web", true, nil, append(webBare, "AI_AGENT=test-agent", "TERM=xterm-256color")},
		// web_environment only reaches web.
		{"db", false, nil, []string{"AI_AGENT=test-agent"}},
		{"db", true, nil, []string{"AI_AGENT=test-agent", "TERM=xterm-256color"}},
		{"web", true, []string{"AI_AGENT=caller", "TERM=vt100", "DDEV_TEST_GLOBAL_BARE=caller", "XDEBUG_MODE=off"}, []string{"DDEV_TEST_PROJECT_BARE=project", "DDEV_TEST_PROJECT_REBARED=host", "DDEV_TEST_BOTH_BARE=both", "DDEV_TEST_IN_DB_FILE=host", "AI_AGENT=caller", "TERM=vt100", "DDEV_TEST_GLOBAL_BARE=caller", "XDEBUG_MODE=off"}},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.expected, app.execEnv(tc.service, tc.tty, tc.callerEnv), "service=%s tty=%v callerEnv=%v", tc.service, tc.tty, tc.callerEnv)
	}
}
