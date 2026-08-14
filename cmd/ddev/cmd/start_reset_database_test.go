package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestCmdStartResetDatabase covers `ddev start --reset-database` and
// `--seed-snapshot`: throwing the database away and starting over, seeding the
// new volume from a named snapshot, and the flag combinations that are refused.
func TestCmdStartResetDatabase(t *testing.T) {
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

	createMarkerTable(t, app)
	require.True(t, tableExists())

	// A snapshot of the marker table, to seed a new volume with further down.
	_, err = app.Snapshot("reset-test-seed")
	require.NoError(t, err)

	// --omit-snapshot means nothing on its own.
	out, err := exec.RunHostCommand(DdevBin, "start", "--omit-snapshot", "-y")
	require.Error(t, err, "output=%s", out)
	require.Contains(t, out, "--reset-database")

	// --reset-database throws a database away, so it can't apply to every project.
	out, err = exec.RunHostCommand(DdevBin, "start", "--all", "--reset-database", "-y")
	require.Error(t, err, "output=%s", out)
	require.Contains(t, out, "--all")

	// The reset itself: the marker table is gone, and a snapshot was taken.
	out, err = exec.RunHostCommand(DdevBin, "start", "--reset-database", "-y")
	require.NoError(t, err, "output=%s", out)
	require.Contains(t, out, "Removed database volume")
	require.False(t, tableExists(), "the database should be back to the stock starter database")

	snapshots, err := app.ListSnapshotNames()
	require.NoError(t, err)
	require.Greater(t, len(snapshots), 1, "the removed database should have been snapshotted: %v", snapshots)

	// --seed-snapshot on a project that already has a database is refused rather
	// than silently ignored.
	out, err = exec.RunHostCommand(DdevBin, "start", "--seed-snapshot=reset-test-seed")
	require.Error(t, err, "output=%s", out)
	require.Contains(t, out, "brand-new database volume")

	// Reset and seed in one command: the marker table comes back.
	out, err = exec.RunHostCommand(DdevBin, "start", "--reset-database", "--omit-snapshot", "--seed-snapshot=reset-test-seed", "-y")
	require.NoError(t, err, "output=%s", out)
	require.Contains(t, out, "Initializing new database volume from snapshot 'reset-test-seed'")
	require.True(t, tableExists(), "the seed snapshot should have populated the new database volume")
}

// dbListTablesCommand returns a command listing the marker table if it's there.
func dbListTablesCommand(app *ddevapp.DdevApp) string {
	if app.Database.Type == nodeps.Postgres {
		return `psql -t -c "SELECT tablename FROM pg_tables WHERE tablename='reset_marker';"`
	}
	return `echo "SHOW TABLES LIKE 'reset_marker';" | ` + app.GetDBClientCommand() + ` -N`
}

// createMarkerTable puts a table in the database that survives only if the
// database itself does.
func createMarkerTable(t *testing.T, app *ddevapp.DdevApp) {
	t.Helper()
	cmd := `echo "CREATE TABLE reset_marker (id INT);" | ` + app.GetDBClientCommand()
	if app.Database.Type == nodeps.Postgres {
		cmd = `psql -c "CREATE TABLE reset_marker (id INT);"`
	}
	_, _, err := app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: cmd})
	require.NoError(t, err)
}

// TestCmdStartSeedSnapshotFile covers `--seed-snapshot` pointed at a snapshot
// file outside the project, which the db container can only reach through a
// mount added for that start.
func TestCmdStartSeedSnapshotFile(t *testing.T) {
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
	createMarkerTable(t, app)

	snapshotName, err := app.Snapshot("outside-seed")
	require.NoError(t, err)
	snapshotFile, err := ddevapp.GetSnapshotFileFromName(snapshotName, app)
	require.NoError(t, err)

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

	out, err := exec.RunHostCommand(DdevBin, "start", "--reset-database", "--omit-snapshot", "--seed-snapshot="+outside, "-y")
	require.NoError(t, err, "output=%s", out)

	tables, _, err := app.Exec(&ddevapp.ExecOpts{Service: "db", Cmd: dbListTablesCommand(app)})
	require.NoError(t, err)
	require.NotEmpty(t, tables, "the out-of-project seed snapshot should have populated the new database volume")
}
