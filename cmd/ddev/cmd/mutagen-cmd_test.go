package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/testcommon"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdMutagen tests `ddev mutagen` config and subcommands
func TestCmdMutagen(t *testing.T) {
	assert := asrt.New(t)
	// Gather reporting about goroutines at exit
	_ = os.Setenv("DDEV_GOROUTINES", "true")

	origDdevDebug := os.Getenv("DDEV_DEBUG")
	_ = os.Setenv(`DDEV_DEBUG`, `true`) // test requires DDEV_DEBUG to see removal of docker volume

	if nodeps.PerformanceModeDefault == types.PerformanceModeMutagen || nodeps.NoBindMountsDefault {
		t.Skip("Skipping because Mutagen on by default")
	}

	site := TestSites[0]
	origDir, _ := os.Getwd()

	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.GetActiveApp("")
	require.NoError(t, err)
	err = app.Stop(true, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		_, err = exec.RunHostCommand(DdevBin, "config", "--performance-mode-reset")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "config", "global", "--performance-mode-reset")
		assert.NoError(err)

		err = globalconfig.ReadGlobalConfig()
		assert.NoError(err)
		app, err = ddevapp.NewApp(site.Dir, true)
		assert.NoError(err)

		_ = os.Setenv(`DDEV_DEBUG`, origDdevDebug)

		err = app.Start()
		assert.NoError(err)

		require.Equal(t, runtime.GOOS == "darwin" || runtime.GOOS == "windows", globalconfig.DdevGlobalConfig.IsMutagenEnabled())
		require.Equal(t, (runtime.GOOS == "darwin" || runtime.GOOS == "windows") && nodeps.PerformanceModeDefault != types.PerformanceModeNFS, app.IsMutagenEnabled())

		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	require.Equal(t, runtime.GOOS == "darwin" || runtime.GOOS == "windows", globalconfig.DdevGlobalConfig.IsMutagenEnabled())
	require.Equal(t, (runtime.GOOS == "darwin" || runtime.GOOS == "windows") && nodeps.PerformanceModeDefault != types.PerformanceModeNFS, app.IsMutagenEnabled())

	// Turn Mutagen off globally
	out, err := exec.RunHostCommand(DdevBin, "config", "global", "--performance-mode=none")
	assert.NoError(err)
	testcommon.CheckGoroutineOutput(t, out)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	app, err = ddevapp.GetActiveApp("")
	require.NoError(t, err)

	require.False(t, globalconfig.DdevGlobalConfig.IsMutagenEnabled())
	require.False(t, app.IsMutagenEnabled())

	// Project override, turn Mutagen on for the project
	_, err = exec.RunHostCommand(DdevBin, "config", "--performance-mode=mutagen")
	assert.NoError(err)

	// Have to reload the app, since we have changed the config
	app, err = ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Make sure it got turned on
	assert.True(app.IsMutagenEnabled())

	t.Logf("DDEV_GOROUTINES before app.StartAndWait()=%s", os.Getenv(`DDEV_GOROUTINES`))

	// Now test subcommands. Wait a bit for Mutagen to get completely done, with transition problems sorted out
	err = app.StartAndWait(10)
	require.NoError(t, err)
	t.Logf("DDEV_GOROUTINES before first mutagen status --verbose=%s", os.Getenv(`DDEV_GOROUTINES`))
	out, err = exec.RunHostCommand(DdevBin, "mutagen", "status", "--verbose")
	testcommon.CheckGoroutineOutput(t, out)

	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "Mutagen: ok"), "expected Mutagen: ok. Full output: %s", out)
	assert.Contains(out, "Mutagen: ok")
	t.Logf("DDEV_GOROUTINES before second mutagen status --verbose=%s", os.Getenv(`DDEV_GOROUTINES`))
	out, err = exec.RunHostCommand(DdevBin, "mutagen", "status", "--verbose")
	assert.NoError(err)
	assert.Contains(out, "Alpha:")
	testcommon.CheckGoroutineOutput(t, out)

	out, err = exec.RunHostCommand(DdevBin, "mutagen", "reset")
	assert.NoError(err)
	assert.Contains(out, fmt.Sprintf("Removed Docker volume %s", ddevapp.GetMutagenVolumeName(app)))
	testcommon.CheckGoroutineOutput(t, out)

	status, statusDesc := app.SiteStatus()
	assert.Equal("stopped", status)
	assert.Equal("stopped", statusDesc)
	err = app.Start()
	assert.NoError(err)
	_, err = exec.RunHostCommand(DdevBin, "mutagen", "sync")
	assert.NoError(err)
	testcommon.CheckGoroutineOutput(t, out)

	err = app.Stop(true, false)
	require.NoError(t, err)
	testcommon.CheckGoroutineOutput(t, out)

	// Turn Mutagen off again
	_, err = exec.RunHostCommand(DdevBin, "config", "--performance-mode-reset")
	require.NoError(t, err)

	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned off
	assert.False(app.IsMutagenEnabled())

	_, err = exec.RunHostCommand(DdevBin, "config", "global", "--performance-mode=mutagen")
	assert.NoError(err)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned on if not overridden by test
	assert.Equal(nodeps.PerformanceModeDefault != types.PerformanceModeNFS, app.IsMutagenEnabled())

	// Turn it off again
	_, err = exec.RunHostCommand(DdevBin, "config", "global", "--performance-mode=none")
	require.NoError(t, err)
	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned off
	assert.False(app.IsMutagenEnabled())
}
