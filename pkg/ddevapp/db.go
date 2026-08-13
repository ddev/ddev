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
)

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
