package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmdSnapshotRestore runs `ddev snapshot restore` on the test apps
func TestCmdSnapshotRestore(t *testing.T) {
	assert := asrt.New(t)
	// Gather reporting about goroutines at exit
	t.Setenv("DDEV_GOROUTINES", "true")

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	assert.NoError(err)
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Make sure all databases are back to default empty
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		_ = os.RemoveAll(app.GetConfigPath("db_snapshots"))
	})

	err = app.Restart()
	require.NoError(t, err)

	// Ensure that a snapshot is created
	args := []string{"snapshot", "--name", "test-snapshot"}
	out, err := exec.RunCommand(DdevBin, args)
	assert.NoError(err)
	assert.Contains(out, "Created database snapshot test-snapshot")
	testcommon.CheckGoroutineOutput(t, out)

	// Ensure that latest snapshot can be restored
	out, err = exec.RunHostCommand(DdevBin, "snapshot", "restore", "--latest")
	assert.NoError(err)
	assert.Contains(out, "Database snapshot test-snapshot was restored")
	testcommon.CheckGoroutineOutput(t, out)
}

// TestCmdSnapshotRestoreFile covers `ddev snapshot restore` pointed at a snapshot
// file outside the project, which the db container can only reach through a mount
// added for that restore.
func TestCmdSnapshotRestoreFile(t *testing.T) {
	origDir, _ := os.Getwd()
	site := TestSites[0]
	require.NoError(t, os.Chdir(site.Dir))
	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, app.Stop(true, false))
		_ = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		require.NoError(t, os.Chdir(origDir))
		require.NoError(t, app.Start())
	})

	require.NoError(t, app.Restart())

	tableExists := func() bool {
		out, _, execErr := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     dbListTablesCommand(app),
		})
		require.NoError(t, execErr)
		return len(out) > 0
	}

	// Snapshot before the marker table exists, so a successful restore removes it.
	snapshotName, err := app.Snapshot("outside-restore")
	require.NoError(t, err)
	snapshotFile, err := ddevapp.GetSnapshotFileFromName(snapshotName, app)
	require.NoError(t, err)

	createMarkerTable(t, app)
	require.True(t, tableExists())

	// Move the snapshot out of the project, where only a mount can reach it.
	// CreateTmpDir keeps it under the home directory, which Docker can bind-mount.
	outsideDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() {
		_ = os.RemoveAll(outsideDir)
	})
	outside := filepath.Join(outsideDir, snapshotFile)
	inProject := app.GetConfigPath(filepath.Join("db_snapshots", snapshotFile))
	require.NoError(t, fileutil.CopyFile(inProject, outside))
	require.NoError(t, os.Remove(inProject))

	out, err := exec.RunHostCommand(DdevBin, "snapshot", "restore", outside)
	require.NoError(t, err, "output=%s", out)
	require.Contains(t, out, "was restored")
	require.False(t, tableExists(), "the out-of-project snapshot should have restored the database to its pre-marker state")
}
