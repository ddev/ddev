package ddevapp_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestSafeChainEnable tests that safe_chain_enable puts the Safe Chain shims
// in front of npm, also when only the global config enables it, and that they
// block a package flagged as malware
func TestSafeChainEnable(t *testing.T) {
	site := TestSites[0]
	origDir, _ := os.Getwd()
	_ = os.Chdir(site.Dir)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	app, err := ddevapp.NewApp(site.Dir, true)
	require.NoError(t, err)
	origSafeChainEnable := app.SafeChainEnable
	origGlobalSafeChainEnable := globalconfig.DdevGlobalConfig.SafeChainEnable
	t.Cleanup(func() {
		runTime()
		_ = os.Chdir(origDir)
		app.SafeChainEnable = origSafeChainEnable
		globalconfig.DdevGlobalConfig.SafeChainEnable = origGlobalSafeChainEnable
		_ = app.Stop(true, false)
	})

	app.SafeChainEnable = false
	globalconfig.DdevGlobalConfig.SafeChainEnable = false
	err = app.Start()
	require.NoError(t, err)
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `command -v npm`,
	})
	require.NoError(t, err)
	require.NotContains(t, out, "safe-chain")
	webDockerfile := app.GetConfigPath(".webimageBuild/Dockerfile")
	found, err := fileutil.FgrepStringInFile(webDockerfile, "Aikido Safe Chain")
	require.NoError(t, err)
	require.False(t, found)

	globalconfig.DdevGlobalConfig.SafeChainEnable = true
	err = app.Start()
	require.NoError(t, err)
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `command -v npm`,
	})
	require.NoError(t, err)
	require.Equal(t, "/usr/local/safe-chain/shims/npm\n", out)
	found, err = fileutil.FgrepStringInFile(webDockerfile, "Aikido Safe Chain")
	require.NoError(t, err)
	require.True(t, found)
	globalconfig.DdevGlobalConfig.SafeChainEnable = false

	app.SafeChainEnable = true
	err = app.Start()
	require.NoError(t, err)

	// Login shells source /etc/profile, which resets PATH
	for _, cmd := range []string{`command -v npm yarn pnpm`, `bash -lc 'command -v npm yarn pnpm'`} {
		out, _, err = app.Exec(&ddevapp.ExecOpts{
			Cmd: cmd,
		})
		require.NoError(t, err)
		require.Equal(t, "/usr/local/safe-chain/shims/npm\n/usr/local/safe-chain/shims/yarn\n/usr/local/safe-chain/shims/pnpm\n", out, "unexpected output from %s", cmd)
	}
	// The bashrc snippet must move the shims to the front, not add them twice
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `bash -lc 'echo "$PATH" | tr : "\n" | grep -c "^/usr/local/safe-chain/shims$"'`,
	})
	require.NoError(t, err)
	require.Equal(t, "1\n", out)
	// Without bash, only ENV PATH from the Dockerfile applies
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		RawCmd: []string{"npm", "safe-chain-verify"},
	})
	require.NoError(t, err)
	require.Contains(t, out, "OK: Safe-chain works!")

	// safe-chain-test is Aikido's test package, always reported as malware;
	// outside the project, so nothing is left behind for later tests
	out, stderr, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: `mkdir -p /tmp/safe-chain-test && cd /tmp/safe-chain-test && echo '{}' >package.json && npm install safe-chain-test </dev/null`,
	})
	require.Error(t, err)
	require.Contains(t, out+stderr, "safe-chain-test")
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `test ! -e /tmp/safe-chain-test/node_modules/safe-chain-test && echo blocked`,
	})
	require.NoError(t, err)
	require.Equal(t, "blocked\n", out)

	// A clean package still installs, so the proxy and its CA work and the
	// project user can write the malware list cache
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Cmd: `cd /tmp/safe-chain-test && npm install is-number@7 </dev/null && test -d node_modules/is-number && echo installed`,
	})
	require.NoError(t, err)
	require.Contains(t, out, "installed")
}
