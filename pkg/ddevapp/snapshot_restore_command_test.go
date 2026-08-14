package ddevapp

import (
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestSnapshotDBVersionFromFilename verifies the db type/version parsed out of a
// snapshot file name, which is what stops a snapshot from being restored into,
// or seeded into, a database it was not made from.
func TestSnapshotDBVersionFromFilename(t *testing.T) {
	for filename, expected := range map[string]string{
		"mysnapshot-mariadb_11.8.zst":               "mariadb_11.8",
		"mysnapshot-mysql_8.0.gz":                   "mysql_8.0",
		"seed-postgres_17.zst":                      "postgres_17",
		"name-with-mariadb_10.11-mariadb_10.11.zst": "mariadb_10.11",
	} {
		dbVersion, err := snapshotDBVersionFromFilename(filename)
		require.NoError(t, err, filename)
		require.Equal(t, expected, dbVersion, filename)
	}

	for _, filename := range []string{"mysnapshot.zst", "mysnapshot-mariadb_11.8.tar", "mariadb_11.8"} {
		_, err := snapshotDBVersionFromFilename(filename)
		require.Error(t, err, filename)
	}

	require.Equal(t, "mariadb", dbTypeFromVersion("mariadb_11.8"))
	require.Equal(t, "postgres", dbTypeFromVersion("postgres_17"))
}

// TestSnapshotRestoreContainerCommand verifies the db container command used
// both by `ddev snapshot restore` and by seeding a brand-new database volume.
// The PostgreSQL command has to create its data directory, which does not exist
// yet on a brand-new volume under PostgreSQL 18+.
func TestSnapshotRestoreContainerCommand(t *testing.T) {
	mariadb := &DdevApp{Name: "test"}
	mariadb.Database = DatabaseDesc{Type: nodeps.MariaDB, Version: nodeps.MariaDB1011}
	require.Equal(t, RestoreSnapshotCommand+" mysnapshot-mariadb_10.11.zst", mariadb.snapshotRestoreContainerCommand("mysnapshot-mariadb_10.11.zst"))
	// An absolute path is passed through, for a seed snapshot mounted outside
	// /mnt/snapshots.
	require.Equal(t, RestoreSnapshotCommand+" /mnt/seed/outside-mariadb_10.11.zst", mariadb.snapshotRestoreContainerCommand("/mnt/seed/outside-mariadb_10.11.zst"))

	for _, pgVersion := range []string{nodeps.Postgres14, nodeps.Postgres18} {
		pg := &DdevApp{Name: "test"}
		pg.Database = DatabaseDesc{Type: nodeps.Postgres, Version: pgVersion}
		command := pg.snapshotRestoreContainerCommand("seed-postgres_" + pgVersion + ".zst")
		require.Contains(t, command, "mkdir -p "+pg.GetPostgresDataDir(), "PostgreSQL %s must create its data directory to seed an empty volume", pgVersion)
		require.Contains(t, command, "/mnt/snapshots/seed-postgres_"+pgVersion+".zst")
		require.True(t, strings.Index(command, "mkdir -p") < strings.Index(command, "chmod 700"), "the data directory has to exist before it can be chmodded")
	}
}
