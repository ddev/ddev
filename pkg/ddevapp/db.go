package ddevapp

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/api/types/container"
)

// DatabaseMismatchMessage describes a project configured for one database while
// its database volume holds another, and spells out the two ways forward: put
// the configuration back, or throw the existing database away. Both `ddev start`
// (which refuses) and `ddev config` (which only warns) report it.
func (app *DdevApp) DatabaseMismatchMessage(existingDBType string) string {
	configuredDBType := app.Database.Type + ":" + app.Database.Version
	message := fmt.Sprintf("project %s is configured for database %s, but its database volume holds %s.\nTo keep the existing database, put the configuration back with 'ddev config --database=%s'.", app.Name, configuredDBType, existingDBType, existingDBType)

	// `ddev utility migrate-database` only converts between MariaDB and MySQL.
	convertible := true
	for _, dbType := range []string{existingDBType, configuredDBType} {
		if !slices.Contains([]string{nodeps.MariaDB, nodeps.MySQL}, strings.Split(dbType, ":")[0]) {
			convertible = false
		}
	}
	if convertible {
		message += fmt.Sprintf("\nTo convert the existing database to %s, put the configuration back and then run 'ddev utility migrate-database %s'.", configuredDBType, configuredDBType)
	}

	return message + fmt.Sprintf("\nTo throw the existing database away and start over with %s, use 'ddev start --reset-database', which snapshots it first unless you add --omit-snapshot.\nSee %s", configuredDBType, "https://docs.ddev.com/en/stable/users/extend/database-types/")
}

// ResetDatabaseVolume removes the project's database volumes so that the next
// start creates and seeds a new one. Unless omitSnapshot, the existing database
// is snapshotted first, at the database version that actually created it, which
// is not necessarily the version the project is now configured for.
func (app *DdevApp) ResetDatabaseVolume(omitSnapshot bool) error {
	volumes := []string{}
	for _, volume := range []string{app.GetMariaDBVolumeName(), app.GetPostgresVolumeName()} {
		if dockerutil.VolumeExists(volume) {
			volumes = append(volumes, volume)
		}
	}
	if len(volumes) == 0 {
		return nil
	}

	if !omitSnapshot {
		snapshotName, err := app.SnapshotAtExistingDBVersion("")
		if err != nil {
			return fmt.Errorf("unable to snapshot the existing database before removing it: %v\nUse --omit-snapshot to remove it without a snapshot", err)
		}
		if snapshotName != "" {
			util.Success("Snapshotted the existing database as '%s' before removing it", snapshotName)
		}
	}

	// The volume can't be removed while the db container has it mounted.
	if dbContainer, err := GetContainer(app, "db"); err == nil && dbContainer != nil {
		if err = dockerutil.RemoveContainer(dbContainer.ID); err != nil {
			return fmt.Errorf("failed to remove db container: %v", err)
		}
	}
	for _, volume := range volumes {
		if err := dockerutil.RemoveVolume(volume); err != nil {
			return fmt.Errorf("failed to remove database volume %s: %v", volume, err)
		}
		util.Success("Removed database volume %s", volume)
	}
	return nil
}

// SnapshotAtExistingDBVersion snapshots the database that is in the database
// volume using the database type/version that created it, which can differ from
// the project's configured one after a `database:` change in config.yaml. It
// returns the snapshot name, or an empty string if there is no database to
// snapshot.
func (app *DdevApp) SnapshotAtExistingDBVersion(snapshotName string) (string, error) {
	existingDBType, err := app.GetExistingDBType()
	if err != nil {
		return "", err
	}
	if existingDBType == "" {
		return "", nil
	}

	// Starting the project with the configured version would refuse outright, so
	// stand the db up as whatever the volume actually holds, just long enough to
	// snapshot it.
	if existingDBType != app.Database.Type+":"+app.Database.Version {
		configuredDatabase := app.Database
		dbType, dbVersion, _ := strings.Cut(existingDBType, ":")
		app.Database = DatabaseDesc{Type: dbType, Version: dbVersion}
		defer func() {
			app.Database = configuredDatabase
		}()
		util.Warning("Starting the database as %s to snapshot it, because that is what its volume holds", existingDBType)
	}

	if status, _ := app.SiteStatus(); status != SiteRunning {
		if err = app.Start(); err != nil {
			return "", err
		}
	}
	return app.Snapshot(snapshotName, false)
}

// GetExistingDBType returns type/version like mariadb:10.11 or postgres:13 or "" if no existing volume
// This has to make a Docker container run so is fairly costly.
func (app *DdevApp) GetExistingDBType() (string, error) {
	dbVersionInfo, err := app.getDBVersionFromVolume()
	if err != nil {
		return "", err
	}

	if dbVersionInfo == "" {
		return "", nil
	}

	return GetDBTypeVersionFromString(dbVersionInfo), nil
}

// getDBVersionFromVolumeScript returns a shell script that finds and prints
// whichever known database version file exists:
//   - mariadbVersionFile: MySQL/MariaDB's db_mariadb_version.txt
//   - postgresVersionFile: PostgreSQL <=17's PG_VERSION, sitting directly at the
//     volume root
//   - postgresVersionGlob: PostgreSQL 18+'s PG_VERSION, one level down in a
//     version-specific directory (e.g. .../18/docker/PG_VERSION) - see
//     GetPostgresDataDir/GetPostgresDataPath, since 18+ mounts the volume one
//     level higher than earlier versions did
//
// Callers pass either the utility-container mount points below, or the real
// in-container paths used when reading directly from a running db container.
func getDBVersionFromVolumeScript(mariadbVersionFile, postgresVersionFile, postgresVersionGlob string) string {
	return fmt.Sprintf(`
		# Check MySQL/MariaDB version file
		if [ -f "%[1]s" ]; then
			cat "%[1]s"
			exit 0
		fi

		# Check PostgreSQL version file (pre-18 location)
		if [ -f "%[2]s" ]; then
			cat "%[2]s"
			exit 0
		fi

		# Check PostgreSQL 18+ version files in version-specific directories
		for version_file in %[3]s; do
			if [ -f "$version_file" ]; then
				cat "$version_file"
				exit 0
			fi
		done

		# No version file found
		exit 0
	`, mariadbVersionFile, postgresVersionFile, postgresVersionGlob)
}

// getDBVersionFromVolume inspects the database volume to determine version info
// Returns the raw version string found in the volume, or empty string if none found
func (app *DdevApp) getDBVersionFromVolume() (string, error) {
	// If the db service is already running, read its version file directly via
	// exec instead of mounting the db volume into a separate utility container.
	// This is both cheaper (no extra container to create/pull/remove) and, on
	// providers where a volume can only be attached read-write by one container
	// at a time (Apple Container/socktainer), the only way this can succeed at
	// all: the running db container already holds the volume read-write, so a
	// second attach - read-only or not - fails outright. See #7372.
	dbContainerName := GetContainerName(app, "db")
	if state, err := dockerutil.GetContainerStateByName(dbContainerName); err == nil && state == "running" {
		script := getDBVersionFromVolumeScript(
			"/var/lib/mysql/db_mariadb_version.txt",
			"/var/lib/postgresql/data/PG_VERSION",
			"/var/lib/postgresql/*/docker/PG_VERSION",
		)
		out, stderr, err := dockerutil.Exec(dbContainerName, script, "")
		if err != nil {
			return "", fmt.Errorf("unable to inspect database version/type on running container %s: %v, stderr=%s", dbContainerName, err, stderr)
		}
		return strings.TrimSpace(out), nil
	}

	// Command to check for database version files:
	// 1. MySQL/MariaDB: /var/tmp/mysql/db_mariadb_version.txt
	// 2. PostgreSQL <=17: /var/tmp/postgres/PG_VERSION
	// 3. PostgreSQL 18+: /var/tmp/postgres/[version]/docker/PG_VERSION (check common versions)
	cmd := []string{"sh", "-c", getDBVersionFromVolumeScript(
		"/var/tmp/mysql/db_mariadb_version.txt",
		"/var/tmp/postgres/PG_VERSION",
		"/var/tmp/postgres/*/docker/PG_VERSION",
	)}

	var volumesNeeded []string
	if dockerutil.VolumeExists(app.GetMariaDBVolumeName()) {
		volumesNeeded = append(volumesNeeded, app.GetMariaDBVolumeName()+":/var/tmp/mysql")
	}
	if dockerutil.VolumeExists(app.GetPostgresVolumeName()) {
		volumesNeeded = append(volumesNeeded, app.GetPostgresVolumeName()+":/var/tmp/postgres")
	}
	// No database volumesNeeded exist
	if len(volumesNeeded) == 0 {
		return "", nil
	}

	// If the project's own db container already has the volume mounted, read it from
	// there instead of mounting it a second time. Cheaper than starting a container,
	// and required on providers that allow a volume only one writer at a time: on
	// apple container the second attachment aborts the probe container's boot with
	// VZErrorDomain Code=2 "The storage device attachment is invalid.", which
	// util.Failed() below turns into a fatal error for the whole command.
	if v, ok := app.dbVersionFromRunningContainer(); ok {
		return v, nil
	}

	_, out, err := dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"GetExistingDBType-"+app.Name+"-"+util.RandString(6),
		cmd,
		[]string{}, // envVars
		[]string{}, // uid
		volumesNeeded,
		"",    // workingDir
		true,  // rm
		false, // detach
		map[string]string{`com.ddev.site-name`: ""}, // labels
		nil, // networks
		nil, // healthcheck
	)

	if err != nil {
		util.Failed("Failed to RunSimpleContainer to inspect database version/type: %v, output=%s", err, out)
	}

	return strings.TrimSpace(out), nil
}

// runningDBContainer returns the project's db container if it is running, else nil.
// Callers use it to avoid mounting the database volume a second time: providers that
// back volumes with a block device (apple container) permit either one writer or any
// number of readers, never a mix, so a second attachment fails while the db runs.
func (app *DdevApp) runningDBContainer() *container.Summary {
	c, err := app.FindContainerByType("db")
	if err != nil || c == nil || c.State != "running" {
		return nil
	}
	return c
}

// dbVersionFromRunningContainer reads the version file out of the project's running
// db container, which already has the database volume mounted. Returns ok=false if
// there is no running db container or the file could not be read, so the caller can
// fall back to the throwaway probe container.
func (app *DdevApp) dbVersionFromRunningContainer() (string, bool) {
	c := app.runningDBContainer()
	if c == nil {
		return "", false
	}

	// Same files the probe container looks for, at the paths the db container itself
	// mounts them on. The mysql path is fixed; the postgres one is version-dependent.
	paths := []string{"/var/lib/mysql/db_mariadb_version.txt"}
	if dataDir := app.GetPostgresDataDir(); dataDir != "" {
		paths = append(paths, dataDir+"/PG_VERSION", app.GetPostgresDataPath()+"/PG_VERSION")
	}

	for _, p := range paths {
		stdout, _, err := dockerutil.Exec(c.ID, "cat "+p, "")
		if err != nil {
			continue
		}
		if v := strings.TrimSpace(stdout); v != "" {
			return v, true
		}
	}

	return "", false
}

// GetDBTypeVersionFromString takes an input string and derives the info from the uses
// There are 3 possible cases here:
// 1. It has an _, meaning it's a current MySQL or MariaDB version. Easy to parse.
// 2. It has N+.N, meaning it's a pre-v1.19 MariaDB or MySQL version
// 3. It has N+, meaning it's PostgreSQL
func GetDBTypeVersionFromString(in string) string {
	idType := ""

	postgresStyle := regexp.MustCompile(`^[0-9]+$`)
	postgresV9Style := regexp.MustCompile(`^9\.?`)
	oldStyle := regexp.MustCompile(`^[0-9]+\.[0-9]$`)
	newStyleV119 := regexp.MustCompile(`^(mysql|mariadb)_[0-9]+\.[0-9][0-9]?$`)

	if newStyleV119.MatchString(in) {
		idType = "current"
	} else if postgresStyle.MatchString(in) || postgresV9Style.MatchString(in) {
		idType = "postgres"
	} else if oldStyle.MatchString(in) {
		idType = "old_pre_v1.19"
	}

	dbType := ""
	dbVersion := ""

	switch idType {
	case "current": // Current representation, <type>_version
		res := strings.Split(in, "_")
		dbType = res[0]
		dbVersion = res[1]

	// PostgreSQL: value is an int
	case "postgres":
		dbType = nodeps.Postgres
		parts := strings.Split(in, `.`)
		dbVersion = parts[0]

	case "old_pre_v1.19":
		dbType = nodeps.MariaDB

		// Both MariaDB and MySQL have 5.5, but we'll give the win to MariaDB here.
		if in == "5.6" || in == "5.7" || in == "8.0" {
			dbType = nodeps.MySQL
		}
		dbVersion = in

	default: // Punt and assume it's an old default db
		dbType = nodeps.MariaDB
		dbVersion = "10.3"
	}
	return dbType + ":" + dbVersion
}

// GetPostgresDataDir returns the correct PostgreSQL data directory path for the given app version
// PostgreSQL 18+ changed the mount point from /var/lib/postgresql/data to /var/lib/postgresql
func (app *DdevApp) GetPostgresDataDir() string {
	if app.Database.Type != nodeps.Postgres {
		return ""
	}
	v, _ := strconv.Atoi(app.Database.Version)
	if v < 18 {
		return "/var/lib/postgresql/data"
	}
	// Postgres 18+ changed the default mount point
	// See https://github.com/docker-library/postgres/pull/1259
	return "/var/lib/postgresql"
}

// GetPostgresDataPath returns the path where PostgreSQL actually stores data files
// This differs from GetPostgresDataDir for PostgreSQL 18+ where files are in a version-specific subdirectory
func (app *DdevApp) GetPostgresDataPath() string {
	if app.Database.Type != nodeps.Postgres {
		return ""
	}

	if slices.Contains([]string{nodeps.Postgres9, nodeps.Postgres10, nodeps.Postgres11, nodeps.Postgres12, nodeps.Postgres13, nodeps.Postgres14, nodeps.Postgres15, nodeps.Postgres16, nodeps.Postgres17}, app.Database.Version) {
		return "/var/lib/postgresql/data"
	}

	// Postgres 18+ stores actual data files in version-specific subdirectory
	// See https://github.com/docker-library/postgres/pull/1259
	return "/var/lib/postgresql/" + app.Database.Version + "/docker"
}
