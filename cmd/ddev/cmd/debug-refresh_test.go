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

// TestDebugRefreshCmd tests that ddev debug refresh actually clears docker caache
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
		err := os.RemoveAll(tmpdir)
		assert.NoError(err)
	})

	err = fileutil.AppendStringToFile(app.GetConfigPath("web-build/Dockerfile"), `
ARG BASE_IMAGE
FROM $BASE_IMAGE
RUN shuf -i 0-99999 -n1 > /random.txt
`)
	require.NoError(t, err)

	// This is normally done in root's init() - probably shouldn't be.
	err = ddevapp.PopulateExamplesCommandsHomeadditions("")
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	origRandom, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random.txt",
	})
	require.NoError(t, err)

	// Make sure that in the ordinary case, the original cache/Dockerfile is same
	err = app.Restart()
	require.NoError(t, err)
	newRandom, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random.txt",
	})
	require.NoError(t, err)
	assert.Equal(origRandom, newRandom)

	// Now run ddev debug refresh to blow away the docker cache
	_, err = exec.RunHostCommand(DdevBin, "debug", "refresh")
	require.NoError(t, err)

	// Now with refresh having been done, we should see a new value for random
	err = app.Restart()
	require.NoError(t, err)
	freshRandom, _, err := app.Exec(&ddevapp.ExecOpts{
		Cmd: "cat /random.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(origRandom, freshRandom)
}
