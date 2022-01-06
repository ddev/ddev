package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// TestCmdSnapshot runs `ddev snapshot` on the test apps
func TestCmdSnapshot(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)
	})

	err = app.Start()
	require.NoError(t, err)

	// Ensure that there are no snapshots available before we create one
	_, err = exec.RunHostCommand(DdevBin, "snapshot", "--cleanup", "--yes")
	assert.NoError(err)

	// Ensure that a snapshot can be created
	out, err := exec.RunHostCommand(DdevBin, "snapshot", "--name", "test-snapshot")
	assert.NoError(err)
	require.Contains(t, out, "Created database snapshot test-snapshot")
	snapshotName := "test-snapshot-mariadb_" + app.MariaDBVersion + ".gz"

	// Try to delete a not existing snapshot
	out, err = exec.RunHostCommand(DdevBin, "snapshot", "--name", "not-existing-snapshot", "--cleanup", "--yes")
	assert.Error(err)
	assert.Contains(out, "Failed to delete snapshot")

	// Ensure that an existing snapshot can be deleted
	out, err = exec.RunHostCommand(DdevBin, "snapshot", "--name", snapshotName, "--cleanup", "--yes")
	assert.NoError(err)
	assert.Contains(out, "Deleted database snapshot '"+snapshotName)
}
