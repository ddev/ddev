package cmd

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdAutostart verifies that commands auto-start a stopped project
// instead of failing with "no running container" errors.
func TestCmdAutostart(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Start()
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	// Stop the project so we can verify auto-start behavior
	err = app.Stop(false, false)
	require.NoError(t, err)

	// ddev exec should auto-start the project
	out, err := exec.RunHostCommand(DdevBin, "exec", "pwd")
	assert.NoError(err, "ddev exec should auto-start stopped project, output=%s", out)
	assert.Contains(out, "/var/www/html")

	// Stop again to test ddev ssh
	err = app.Stop(false, false)
	require.NoError(t, err)

	// ddev ssh should auto-start the project; pipe "exit" so the shell exits cleanly
	bash := util.FindBashPath()
	out, err = exec.RunHostCommand(bash, "-c", "echo exit | "+DdevBin+" ssh")
	assert.NoError(err, "ddev ssh should auto-start stopped project, output=%s", out)

	// Verify the app is running now (auto-start left it running)
	status, _ := app.SiteStatus()
	assert.Equal(ddevapp.SiteRunning, status, "app should be running after auto-start by ddev ssh")
}
