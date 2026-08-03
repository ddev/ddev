package ddevapp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// newBaseDBSeedTestApp returns an unstarted app rooted in a temp directory with
// a .ddev directory, enough for the host-side base_db seed lookups.
func newBaseDBSeedTestApp(t *testing.T, dbType, dbVersion string) *DdevApp {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".ddev"), 0755))
	app := &DdevApp{
		Name:       "base-db-seed-test",
		AppRoot:    tmpDir,
		ConfigPath: filepath.Join(tmpDir, ".ddev", "config.yaml"),
	}
	app.Database.Type = dbType
	app.Database.Version = dbVersion
	return app
}

// writeBaseDBSeedTestFile writes content to a path under the project's .ddev directory.
func writeBaseDBSeedTestFile(t *testing.T, app *DdevApp, relPath, content string) string {
	t.Helper()
	fullPath := app.GetConfigPath(relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	return fullPath
}

// TestGetInitializerSnapshotFile verifies the host-side lookup of the reserved
// `initializer` snapshot, which must match ddev-dbserver's docker-entrypoint.sh:
// named for the exact db type/version, preferring .zst over .gz.
func TestGetInitializerSnapshotFile(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	require.Empty(t, app.GetInitializerSnapshotFile())

	gz := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "initializer-mariadb_10.11.gz"), "seed")
	require.Equal(t, gz, app.GetInitializerSnapshotFile())

	// .zst wins over .gz when both are present.
	zst := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "initializer-mariadb_10.11.zst"), "seed")
	require.Equal(t, zst, app.GetInitializerSnapshotFile())

	// A snapshot for a different db version is not this project's initializer.
	other := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB106)
	writeBaseDBSeedTestFile(t, other, filepath.Join("db_snapshots", "initializer-mariadb_10.11.zst"), "seed")
	require.Empty(t, other.GetInitializerSnapshotFile())

	// PostgreSQL doesn't use base_db seeding at all.
	pg := newBaseDBSeedTestApp(t, nodeps.Postgres, nodeps.Postgres17)
	writeBaseDBSeedTestFile(t, pg, filepath.Join("db_snapshots", "initializer-postgres_17.zst"), "seed")
	require.Empty(t, pg.GetInitializerSnapshotFile())

	// Neither does a project with no db container.
	omitted := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	omitted.OmitContainers = []string{"db"}
	writeBaseDBSeedTestFile(t, omitted, filepath.Join("db_snapshots", "initializer-mariadb_10.11.zst"), "seed")
	require.Empty(t, omitted.GetInitializerSnapshotFile())
}

// TestGetDBBuildSeedDockerfiles verifies that only db-build Dockerfiles that
// actually bake a base_db seed into the dbimage are reported, and that the
// bundled Dockerfile.example is ignored.
func TestGetDBBuildSeedDockerfiles(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)

	writeBaseDBSeedTestFile(t, app, filepath.Join("db-build", "Dockerfile"), "RUN echo hi\n")
	require.Empty(t, app.GetDBBuildSeedDockerfiles())
	require.True(t, app.mayHaveDerivedDBImageSeed(), "a db-build Dockerfile makes a derived-image seed possible")

	seed := writeBaseDBSeedTestFile(t, app, filepath.Join("db-build", "Dockerfile.seed"), "COPY base_db.zst /mysqlbase/custom/base_db.zst\n")
	require.Equal(t, []string{seed}, app.GetDBBuildSeedDockerfiles())

	// Dockerfile.example ships in every project and is never built, so it must
	// not make us think a derived image (or a seed) is in play.
	clean := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	writeBaseDBSeedTestFile(t, clean, filepath.Join("db-build", "Dockerfile.example"), "COPY base_db.zst /mysqlbase/custom/base_db.zst\n")
	require.Empty(t, clean.GetDBBuildSeedDockerfiles())
	require.False(t, clean.mayHaveDerivedDBImageSeed())
}

// TestAnnounceBaseDBSeed verifies that seeding a fresh database volume from a
// project `initializer` snapshot is announced with what it's doing, which
// snapshot, and that it may take a while — and that the stock starter database
// (no initializer, no derived image) stays quiet.
func TestAnnounceBaseDBSeed(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	app.DefaultContainerTimeout = nodeps.DefaultDefaultContainerTimeout

	require.Empty(t, app.BaseDBSeedDescription(), "stock starter database should not be announced")

	initializer := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "initializer-mariadb_10.11.zst"), strings.Repeat("x", 2048))
	description := app.BaseDBSeedDescription()
	require.NotEmpty(t, description)

	outFunc := util.CaptureUserOut()
	app.announceBaseDBSeed(description, SnapshotRestoreDefaultWaitTime)
	out := outFunc()

	require.Contains(t, out, "Initializing new database volume from the 'initializer' snapshot")
	require.Contains(t, out, filepath.ToSlash(initializer))
	require.Contains(t, out, "(2.0KB)")
	require.Contains(t, out, "this may take a long time")
	require.Contains(t, out, "default_container_timeout")
	require.Contains(t, out, "ddev logs -s db -f "+app.Name)
	require.Contains(t, out, strconv.Itoa(SnapshotRestoreDefaultWaitTime))
}

// TestDerivedBuiltImageRef verifies the "-<project>-built" image reference
// getDerivedDBImageSeed() checks always carries an explicit tag, since
// dockerutil.RunSimpleContainer rejects a bare repo with no ":tag" outright.
// A `dbimage:` set without an explicit tag (e.g. "randyfay/dbserver-2m") is a
// common way to configure it, and previously produced an untagged
// "repo-project-built" reference that RunSimpleContainer flatly rejected,
// silently swallowed by util.Debug -- see base_db_seed.go.
func TestDerivedBuiltImageRef(t *testing.T) {
	// No tag on the base image: matches what `docker compose build` actually
	// tags it as, since Docker itself defaults a tag-less repo to ":latest".
	require.Equal(t, "randyfay/dbserver-2m-myproject-built:latest", derivedBuiltImageRef("randyfay/dbserver-2m", "myproject"))

	// An explicit tag already present: the suffix lands in the tag portion,
	// same convention used for app.WebImage elsewhere in this package.
	require.Equal(t, "ddev/ddev-dbserver-mariadb-11.8:v1.25.3-myproject-built", derivedBuiltImageRef("ddev/ddev-dbserver-mariadb-11.8:v1.25.3", "myproject"))
}

// TestFileSizeSuffix verifies the human-readable size suffix used when
// announcing an `initializer` snapshot restore.
func TestFileSizeSuffix(t *testing.T) {
	tmpDir := t.TempDir()

	small := filepath.Join(tmpDir, "small")
	require.NoError(t, os.WriteFile(small, make([]byte, 512), 0644))
	require.Equal(t, " (512B)", fileSizeSuffix(small))

	large := filepath.Join(tmpDir, "large")
	require.NoError(t, os.WriteFile(large, make([]byte, 3*1024*1024), 0644))
	require.Equal(t, " (3.0MB)", fileSizeSuffix(large))

	require.Empty(t, fileSizeSuffix(filepath.Join(tmpDir, "nonexistent")))
	require.Empty(t, fileSizeSuffix(tmpDir))
}
