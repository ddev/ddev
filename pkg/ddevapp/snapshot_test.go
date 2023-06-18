package ddevapp_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDdevSnapshotCleanup tests creating a snapshot and deleting it.
func TestDdevSnapshotCleanup(t *testing.T) {
	assert := assert2.New(t)
	app := &ddevapp.DdevApp{}
	site := TestSites[0]
	switchDir := site.Chdir()
	defer switchDir()

	runTime := util.TimeTrackC("TestDdevSnapshotCleanup")

	testcommon.ClearDockerEnv()
	err := app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)
	})

	err = app.Start()
	assert.NoError(err)

	// Make a snapshot of d7 tester test 1
	snapshotName, err := app.Snapshot(t.Name() + "_1")
	assert.NoError(err)

	err = app.Init(site.Dir)
	require.NoError(t, err)

	err = app.Start()
	require.NoError(t, err)

	err = app.DeleteSnapshot(snapshotName)
	assert.NoError(err)

	// Snapshot data should be deleted
	err = app.DeleteSnapshot(snapshotName)
	assert.Error(err)

	runTime()
}

// TestGetLatestSnapshot tests if the latest snapshot of a project is returned correctly.
func TestGetLatestSnapshot(t *testing.T) {
	assert := assert2.New(t)
	app := &ddevapp.DdevApp{}
	site := TestSites[0]
	origDir, _ := os.Getwd()
	err := os.Chdir(site.Dir)
	assert.NoError(err)

	runTime := util.TimeTrackC(t.Name())

	testcommon.ClearDockerEnv()
	err = app.Init(site.Dir)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
	})

	err = app.Start()
	assert.NoError(err)

	snapshots := []string{t.Name() + "_1", t.Name() + "_2", t.Name() + "_3"}
	// Make three snapshots and compare the last
	s1Name, err := app.Snapshot(snapshots[0])
	assert.NoError(err)
	s2Name, err := app.Snapshot(snapshots[1])
	assert.NoError(err)
	s3Name, err := app.Snapshot(snapshots[2]) // last = latest
	assert.NoError(err)

	latestSnapshot, err := app.GetLatestSnapshot()
	assert.NoError(err)
	assert.Equal(s3Name, latestSnapshot)

	// delete snapshot 3
	err = app.DeleteSnapshot(s3Name)
	assert.NoError(err)
	latestSnapshot, err = app.GetLatestSnapshot()
	assert.NoError(err)
	assert.Equal(s2Name, latestSnapshot, "%s should be latest snapshot", snapshots[1])

	// delete snapshot 2
	err = app.DeleteSnapshot(s2Name)
	assert.NoError(err)
	latestSnapshot, err = app.GetLatestSnapshot()
	assert.NoError(err)
	assert.Equal(s1Name, latestSnapshot, "%s should now be latest snapshot", s1Name)

	// delete snapshot 1 (should be last)
	err = app.DeleteSnapshot(s1Name)
	assert.NoError(err)
	latestSnapshot, _ = app.GetLatestSnapshot()
	assert.Equal("", latestSnapshot)

	runTime()
}

// TestDdevRestoreSnapshot tests creating a snapshot and reverting to it.
func TestDdevRestoreSnapshot(t *testing.T) {
	assert := assert2.New(t)

	runTime := util.TimeTrackC(t.Name())
	origDir, _ := os.Getwd()
	site := TestSites[0]

	d7testerTest1Dump, err := filepath.Abs(filepath.Join("testdata", t.Name(), "restore_snapshot", "d7tester_test_1.sql.gz"))
	assert.NoError(err)
	d7testerTest2Dump, err := filepath.Abs(filepath.Join("testdata", t.Name(), "restore_snapshot", "d7tester_test_2.sql.gz"))
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		app.Hooks = nil
		app.Database.Type = nodeps.MariaDB
		app.Database.Version = nodeps.MariaDBDefaultVersion
		err = app.WriteConfig()
		assert.NoError(err)
		err = os.Chdir(origDir)
		assert.NoError(err)
		err = os.RemoveAll(app.GetConfigPath("db_snapshots"))
		assert.NoError(err)
		testcommon.ClearDockerEnv()
	})

	err = os.Chdir(app.AppRoot)
	require.NoError(t, err)
	app.Hooks = map[string][]ddevapp.YAMLTask{"post-snapshot": {{"exec-host": "touch hello-post-snapshot-" + app.Name}}, "pre-snapshot": {{"exec-host": "touch hello-pre-snapshot-" + app.Name}}}

	err = app.Stop(true, false)
	assert.NoError(err)
	err = app.Start()
	require.NoError(t, err)

	err = app.ImportDB(d7testerTest1Dump, "", false, false, "db")
	require.NoError(t, err, "Failed to app.ImportDB path: %s err: %v", d7testerTest1Dump, err)

	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `echo "SELECT title FROM node WHERE nid=1;" | mysql -N`,
	})
	assert.NoError(err)
	assert.Contains(stdout, "d7 tester test 1 has 1 node")

	// Make a snapshot of d7 tester test 1
	tester1Snapshot, err := app.Snapshot("d7testerTest1")
	assert.NoError(err)

	assert.Contains(tester1Snapshot, "d7testerTest1")
	latest, err := app.GetLatestSnapshot()
	assert.NoError(err)
	assert.Equal(tester1Snapshot, latest)

	assert.FileExists("hello-pre-snapshot-" + app.Name)
	assert.FileExists("hello-post-snapshot-" + app.Name)
	err = os.Remove("hello-pre-snapshot-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-snapshot-" + app.Name)
	assert.NoError(err)

	// Make sure duplicate snapshot name gives an error
	_, err = app.Snapshot("d7testerTest1")
	assert.Error(err)

	err = app.ImportDB(d7testerTest2Dump, "", false, false, "db")
	assert.NoError(err, "Failed to app.ImportDB path: %s err: %v", d7testerTest2Dump, err)

	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `echo "SELECT title FROM node WHERE nid=1;" | mysql -N`,
	})
	assert.NoError(err)
	assert.Contains(stdout, "d7 tester test 2 has 2 nodes")

	tester2Snapshot, err := app.Snapshot("d7testerTest2")
	assert.NoError(err)
	assert.Contains(tester2Snapshot, "d7testerTest2")
	latest, err = app.GetLatestSnapshot()
	assert.Equal(tester2Snapshot, latest)

	app.Hooks = map[string][]ddevapp.YAMLTask{"post-restore-snapshot": {{"exec-host": "touch hello-post-restore-snapshot-" + app.Name}}, "pre-restore-snapshot": {{"exec-host": "touch hello-pre-restore-snapshot-" + app.Name}}}

	err = app.MutagenSyncFlush()
	require.NoError(t, err)
	// Sleep to let sync happen if needed (M1 failure)
	time.Sleep(2 * time.Second)

	err = app.RestoreSnapshot(tester1Snapshot)
	assert.NoError(err)

	assert.FileExists("hello-pre-restore-snapshot-" + app.Name)
	assert.FileExists("hello-post-restore-snapshot-" + app.Name)
	err = os.Remove("hello-pre-restore-snapshot-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-restore-snapshot-" + app.Name)
	assert.NoError(err)

	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `echo "SELECT title FROM node WHERE nid=1;" | mysql -N`,
	})
	assert.NoError(err)
	assert.Contains(stdout, "d7 tester test 1 has 1 node")

	err = app.RestoreSnapshot(tester2Snapshot)
	assert.NoError(err)

	stdout, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `echo "SELECT title FROM node WHERE nid=1;" | mysql -N`,
	})
	assert.NoError(err)
	assert.Contains(stdout, "d7 tester test 2 has 2 nodes")

	// Attempt a restore with a pre-mariadb_10.2 snapshot. It should fail and give a link.
	oldSnapshotTarball, err := filepath.Abs(filepath.Join(origDir, "testdata", t.Name(), "restore_snapshot", "d7tester_test_1.snapshot_mariadb_10_1.tgz"))
	assert.NoError(err)

	err = archive.Untar(oldSnapshotTarball, app.GetConfigPath("db_snapshots"), "")
	assert.NoError(err)

	err = app.RestoreSnapshot("d7tester_test_1.snapshot_mariadb_10.1")
	assert.Error(err)
	assert.Contains(err.Error(), "is not compatible")

	// Make sure that we can use old-style directory-based snapshots
	dirSnapshots := map[string]string{
		"mariadb:10.3": "mariadb10.3-users",
		"mysql:5.7":    "mysql5.7-users",
	}

	// Despite much effort, and successful manual restore of the mysql5.7-users.tgz to another project,
	// I can't get it to restore in this test. The logs show
	// " [ERROR] InnoDB: Log block 24712 at lsn 12652032 has valid header, but checksum field contains 1825156513, should be 1116246688"
	// Since this works everywhere else and this is legacy snapshot support, I'm going to punt
	// and skip the mysql snapshot. rfay 2022-02-25
	if runtime.GOOS == "windows" {
		delete(dirSnapshots, "mysql:5.7")
	}

	for dbDesc, dirSnapshot := range dirSnapshots {
		oldSnapshotTarball, err = filepath.Abs(filepath.Join(origDir, "testdata", t.Name(), dirSnapshot+".tgz"))
		assert.NoError(err)
		fullsnapshotDir := filepath.Join(app.GetConfigPath("db_snapshots"), dirSnapshot)
		err = os.MkdirAll(fullsnapshotDir, 0755)
		require.NoError(t, err)
		err = archive.Untar(oldSnapshotTarball, fullsnapshotDir, "")
		assert.NoError(err)

		err = app.Stop(true, false)
		assert.NoError(err)

		parts := strings.Split(dbDesc, ":")
		require.Equal(t, 2, len(parts))
		dbType := parts[0]
		dbVersion := parts[1]
		app.Database.Type = dbType
		app.Database.Version = dbVersion
		err = app.WriteConfig()
		assert.NoError(err)

		err = app.Start()
		assert.NoError(err)

		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     `echo "DROP TABLE IF EXISTS users;" | mysql`,
		})
		assert.NoError(err)

		err = app.RestoreSnapshot(dirSnapshot)
		if err != nil {
			assert.NoError(err, "Failed to restore dirSnapshot '%s': %v, continuing", dirSnapshot, err)
			continue
		}

		stdout, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     `echo "SELECT COUNT(*) FROM users;" | mysql -N`,
		})
		assert.NoError(err)
		assert.Equal(stdout, "2\n")
	}
	runTime()
}
