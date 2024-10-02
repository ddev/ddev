package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
)

// TestDebugRefreshCmd tests that ddev debug refresh actually clears Docker cache
func TestDebugRefreshCmd(t *testing.T) {
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

	origRandomDb, _, err := app.Exec(&ddevapp.ExecOpts{
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

	newRandomDb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(origRandomDb, newRandomDb)

	// Now run ddev debug refresh to blow away the Docker cache
	_, err = exec.RunHostCommand(DdevBin, "debug", "refresh")
	require.NoError(t, err)

	// Now with refresh having been done, we should see a new value for random
	err = app.Restart()
	require.NoError(t, err)
	freshRandomWeb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random-web.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(origRandomWeb, freshRandomWeb)

	// And it should remain the same for db
	freshRandomDb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(origRandomDb, freshRandomDb)

	// Now run ddev debug refresh to blow away the Docker cache for db
	_, err = exec.RunHostCommand(DdevBin, "debug", "refresh", "--service", "db")
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
	freshRandomDbNew, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.NotEqual(freshRandomDb, freshRandomDbNew)

	// Repeat the same with all services, but use cache
	_, err = exec.RunHostCommand(DdevBin, "debug", "refresh", "--all", "--cache")
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
	cachedRandomDb, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd:     "cat /random-db.txt",
		Service: "db",
	})
	require.NoError(t, err)
	assert.Equal(cachedRandomDb, freshRandomDbNew)
}
