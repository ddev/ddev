package cmd

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugRebuildCmd tests that ddev utility rebuild actually clears Docker cache
func TestDebugRebuildCmd(t *testing.T) {
	// Don't run this unless GOTEST_SHORT is unset; it doesn't need to be run everywhere.
	if os.Getenv("GOTEST_SHORT") != "" {
		t.Skip("Skip because GOTEST_SHORT is set")
	}

	assert := asrt.New(t)

	// Create a temporary directory and switch to it.
	origDir, _ := os.Getwd()
	tmpdir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(tmpdir)
	assert.NoError(err)
	_, err = exec.RunHostCommand(DdevBin, "config", "--auto")
	assert.NoError(err)

	app, err := ddevapp.GetActiveApp("")
	assert.NoError(err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		err = app.Stop(true, false)
		assert.NoError(err)
		// On Windows the tmpdir may not be removable for unknown reasons, don't check result
		_ = os.RemoveAll(tmpdir)
	})

	err = fileutil.AppendStringToFile(app.GetConfigPath("web-build/Dockerfile"), `
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN shuf -i 0-99999 -n1 > /random-web.txt
`)
	require.NoError(t, err)

	err = fileutil.AppendStringToFile(app.GetConfigPath("db-build/Dockerfile"), `
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN shuf -i 0-99999 -n1 > /random-db.txt
`)
	require.NoError(t, err)

	// This is normally done in root's init() - probably shouldn't be.
	err = ddevapp.PopulateExamplesCommandsHomeadditions("")
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	origRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)

	origRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)

	// Make sure that in the ordinary case, the original cache/Dockerfile is same
	err = app.Restart()
	require.NoError(t, err)
	newRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	assert.Equal(origRandomWeb, newRandomWeb)

	newRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(origRandomDB, newRandomDB)

	// Now run ddev utility rebuild to blow away the Docker cache
	_, err = exec.RunHostCommand(DdevBin, "utility", "rebuild")
	require.NoError(t, err)

	// Now with rebuild having been done, we should see a new value for random
	err = app.Restart()
	require.NoError(t, err)
	freshRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(origRandomWeb, freshRandomWeb)

	// And it should remain the same for db
	freshRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(origRandomDB, freshRandomDB)

	// Now run ddev utility rebuild to blow away the Docker cache for db
	_, err = exec.RunHostCommand(DdevBin, "utility", "rebuild", "--service", "db")
	require.NoError(t, err)

	// It should remain the same for web
	err = app.Restart()
	require.NoError(t, err)
	freshRandomWebNew, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	assert.Equal(freshRandomWeb, freshRandomWebNew)

	// And we should see a new value for db
	err = app.Restart()
	require.NoError(t, err)
	freshRandomDBNew, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.NotEqual(freshRandomDB, freshRandomDBNew)

	// Repeat the same with all services, but use cache
	_, err = exec.RunHostCommand(DdevBin, "utility", "rebuild", "--all", "--cache")
	require.NoError(t, err)

	// It should remain the same for web
	err = app.Restart()
	require.NoError(t, err)
	cachedRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	assert.Equal(cachedRandomWeb, freshRandomWebNew)

	// And it should remain the same for db
	err = app.Restart()
	require.NoError(t, err)
	cachedRandomDB, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(cachedRandomDB, freshRandomDBNew)
}
