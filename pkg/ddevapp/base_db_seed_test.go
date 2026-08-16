package ddevapp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
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

// TestGetSeedSnapshotFile verifies the lookup of the reserved `seed` snapshot:
// named for the exact db type/version, preferring .zst over .gz.
func TestGetSeedSnapshotFile(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	require.Empty(t, app.GetSeedSnapshotFile())

	gz := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "seed-mariadb_10.11.gz"), "seed")
	require.Equal(t, gz, app.GetSeedSnapshotFile())

	// .zst wins over .gz when both are present.
	zst := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "seed-mariadb_10.11.zst"), "seed")
	require.Equal(t, zst, app.GetSeedSnapshotFile())

	// An uncompressed .mbstream seed is recognized, but a compressed seed
	// still wins if one is also present.
	writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "initializer-mariadb_10.11.mbstream"), "seed")
	require.Equal(t, zst, app.GetInitializerSnapshotFile())

	// With no compressed seed, the uncompressed one is used.
	uncompressed := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	mbstream := writeBaseDBSeedTestFile(t, uncompressed, filepath.Join("db_snapshots", "initializer-mariadb_10.11.mbstream"), "seed")
	require.Equal(t, mbstream, uncompressed.GetInitializerSnapshotFile())

	// A snapshot for a different db version is not this project's initializer.
	other := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB106)
	writeBaseDBSeedTestFile(t, other, filepath.Join("db_snapshots", "seed-mariadb_10.11.zst"), "seed")
	require.Empty(t, other.GetSeedSnapshotFile())

	// Unlike a base_db seed baked into the dbimage, this works for PostgreSQL.
	pg := newBaseDBSeedTestApp(t, nodeps.Postgres, nodeps.Postgres17)
	pgSeed := writeBaseDBSeedTestFile(t, pg, filepath.Join("db_snapshots", "seed-postgres_17.zst"), "seed")
	require.Equal(t, pgSeed, pg.GetSeedSnapshotFile())

	// A project with no db container has nothing to seed.
	omitted := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	omitted.OmitContainers = []string{"db"}
	writeBaseDBSeedTestFile(t, omitted, filepath.Join("db_snapshots", "seed-mariadb_10.11.zst"), "seed")
	require.Empty(t, omitted.GetSeedSnapshotFile())
}

// TestResolveSeedSnapshot covers what `--seed-snapshot` accepts: nothing (the
// reserved `seed` snapshot), a snapshot name in .ddev/db_snapshots, and a path
// to a snapshot elsewhere, which needs a mount of its own.
func TestResolveSeedSnapshot(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)

	origNoBindMounts := globalconfig.DdevGlobalConfig.NoBindMounts
	globalconfig.DdevGlobalConfig.NoBindMounts = false
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.NoBindMounts = origNoBindMounts })

	seed, err := app.ResolveSeedSnapshot()
	require.NoError(t, err)
	require.Nil(t, seed, "nothing to seed from without a flag or a reserved snapshot")

	reserved := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "seed-mariadb_10.11.zst"), "seed")
	seed, err = app.ResolveSeedSnapshot()
	require.NoError(t, err)
	require.Equal(t, reserved, seed.HostPath)
	require.Empty(t, seed.MountDir, "a snapshot in the project needs no mount of its own")
	require.Equal(t, "/mnt/snapshots/seed-mariadb_10.11.zst", seed.ContainerPath())

	// A bare name resolves inside .ddev/db_snapshots, the same way
	// `ddev snapshot restore <name>` resolves it.
	inProject := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "mydata-mariadb_10.11.zst"), "seed")
	app.SeedSnapshot = "mydata"
	seed, err = app.ResolveSeedSnapshot()
	require.NoError(t, err)
	require.Equal(t, inProject, seed.HostPath)
	require.Empty(t, seed.MountDir)

	// A path outside the project is mounted into the db container.
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "elsewhere-mariadb_10.11.zst")
	require.NoError(t, os.WriteFile(outside, []byte("seed"), 0644))
	app.SeedSnapshot = outside
	seed, err = app.ResolveSeedSnapshot()
	require.NoError(t, err)
	require.Equal(t, outside, seed.HostPath)
	require.Equal(t, outsideDir, seed.MountDir)
	require.Equal(t, SeedSnapshotMountDir+"/elsewhere-mariadb_10.11.zst", seed.ContainerPath())

	// With no_bind_mounts there is nothing to mount; prepareSeedSnapshot copies
	// the snapshot into the snapshots volume, so the container path is the usual one.
	globalconfig.DdevGlobalConfig.NoBindMounts = true
	seed, err = app.ResolveSeedSnapshot()
	require.NoError(t, err)
	require.Equal(t, outside, seed.HostPath)
	require.Empty(t, seed.MountDir)
	require.Equal(t, "/mnt/snapshots/elsewhere-mariadb_10.11.zst", seed.ContainerPath())
	globalconfig.DdevGlobalConfig.NoBindMounts = false

	// A snapshot from another database can't seed this one.
	wrongVersion := filepath.Join(outsideDir, "elsewhere-mariadb_10.6.zst")
	require.NoError(t, os.WriteFile(wrongVersion, []byte("seed"), 0644))
	app.SeedSnapshot = wrongVersion
	_, err = app.ResolveSeedSnapshot()
	require.ErrorContains(t, err, "mariadb_10.6")

	app.SeedSnapshot = filepath.Join(outsideDir, "no-such-snapshot.zst")
	_, err = app.ResolveSeedSnapshot()
	require.ErrorContains(t, err, "not an existing snapshot file")

	app.SeedSnapshot = "nonexistent"
	_, err = app.ResolveSeedSnapshot()
	require.ErrorContains(t, err, "nonexistent")
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
// project `seed` snapshot is announced with what it's doing, which snapshot, and
// that it may take a while — and that the stock starter database (no seed
// snapshot, no derived image) stays quiet.
func TestAnnounceBaseDBSeed(t *testing.T) {
	app := newBaseDBSeedTestApp(t, nodeps.MariaDB, nodeps.MariaDB1011)
	app.DefaultContainerTimeout = nodeps.DefaultDefaultContainerTimeout

	require.Empty(t, app.SeedDescription(nil), "stock starter database should not be announced")

	seedFile := writeBaseDBSeedTestFile(t, app, filepath.Join("db_snapshots", "seed-mariadb_10.11.zst"), strings.Repeat("x", 2048))
	seed, err := app.ResolveSeedSnapshot()
	require.NoError(t, err)
	description := app.SeedDescription(seed)
	require.NotEmpty(t, description)

	outFunc := util.CaptureUserOut()
	app.announceBaseDBSeed(description, SnapshotRestoreDefaultWaitTime)
	out := outFunc()

	require.Contains(t, out, "Initializing new database volume from the 'seed' snapshot")
	require.Contains(t, out, fileutil.ShortHomeJoin(filepath.ToSlash(seedFile)))
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
// announcing a seed snapshot restore.
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
