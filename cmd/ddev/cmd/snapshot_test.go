package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestCmdSnapshot runs `ddev snapshot` on the test apps
func TestCmdSnapshot(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	cleanup := site.Chdir()

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		cleanup()
	})

	err = app.Start()
	require.NoError(t, err)

	// Ensure that there are no snapshots available before we create one
	args := []string{"snapshot", "--cleanup", "--yes"}
	_, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)

	// Ensure that a snapshot can be created
	args = []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Created snapshot test-snapshot")

	// Try to delete a not existing snapshot
	args = []string{"snapshot", "--name", "not-existing-snapshot", "--cleanup", "--yes"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.Error(err)
	assert.Contains(out, "Failed to delete snapshot")

	// Ensure that an existing snapshot can be deleted
	args = []string{"snapshot", "--name", "test-snapshot", "--cleanup", "--yes"}
	out, err = exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Deleted database snapshot test-snapshot")
}
