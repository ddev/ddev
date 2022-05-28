package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
)

// TestCmdMutagen tests `ddev mutagen` config and subcommands
func TestCmdMutagen(t *testing.T) {
	assert := asrt.New(t)

	if nodeps.MutagenEnabledDefault || nodeps.NoBindMountsDefault {
		t.Skip("Skipping because mutagen on by default")
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
		_, err = exec.RunHostCommand(DdevBin, "config", "--mutagen-enabled=false")
		assert.NoError(err)
		_, err = exec.RunHostCommand(DdevBin, "config", "global", "--mutagen-enabled=false")
		assert.NoError(err)
		app, err = ddevapp.NewApp(site.Dir, true)
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
		assert.False(app.IsMutagenEnabled())
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	require.False(t, globalconfig.DdevGlobalConfig.MutagenEnabledGlobal)
	require.False(t, app.IsMutagenEnabled())

	_, err = exec.RunHostCommand(DdevBin, "config", "--mutagen-enabled=true")
	assert.NoError(err)

	// Have to reload the app, since we just changed the config
	app, err = ddevapp.GetActiveApp("")
	require.NoError(t, err)

	// Make sure it got turned on
	assert.True(app.IsMutagenEnabled())

	// Now test subcommands. Wait just a bit for mutagen to get completely done, with transition problems sorted outx
	err = app.StartAndWait(10)
	require.NoError(t, err)
	out, err := exec.RunHostCommand(DdevBin, "mutagen", "status", "--verbose")
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "Mutagen: ok"), "expected Mutagen: ok. Full output: %s", out)
	assert.Contains(out, "Mutagen: ok")
	out, err = exec.RunHostCommand(DdevBin, "mutagen", "status", "--verbose")
	assert.NoError(err)
	assert.Contains(out, "Alpha:")

	out, err = exec.RunHostCommand(DdevBin, "mutagen", "reset")
	assert.NoError(err)
	assert.Contains(out, fmt.Sprintf("Removed docker volume %s", ddevapp.GetMutagenVolumeName(app)))

	status, statusDesc := app.SiteStatus()
	assert.Equal("stopped", status)
	assert.Equal("stopped", statusDesc)
	err = app.Start()
	assert.NoError(err)
	_, err = exec.RunHostCommand(DdevBin, "mutagen", "sync")
	assert.NoError(err)

	err = app.Stop(true, false)
	require.NoError(t, err)

	// Turn mutagen off again
	_, err = exec.RunHostCommand(DdevBin, "config", "--mutagen-enabled=false")
	require.NoError(t, err)

	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned off
	assert.False(app.IsMutagenEnabled())

	_, err = exec.RunHostCommand(DdevBin, "config", "global", "--mutagen-enabled=true")
	assert.NoError(err)

	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)
	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned on
	assert.True(app.MutagenEnabledGlobal)

	// Turn it off again
	_, err = exec.RunHostCommand(DdevBin, "config", "global", "--mutagen-enabled=false")
	require.NoError(t, err)
	err = globalconfig.ReadGlobalConfig()
	require.NoError(t, err)

	app, err = ddevapp.NewApp("", false)
	require.NoError(t, err)

	// Make sure it got turned off
	assert.False(app.MutagenEnabledGlobal)
}
