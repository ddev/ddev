package ddevapp

import (
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

// getDBVersionFromVolume inspects the database volume to determine version info
// Returns the raw version string found in the volume, or empty string if none found
func (app *DdevApp) getDBVersionFromVolume() (string, error) {
	// Command to check for database version files:
	// 1. MySQL/MariaDB: /var/tmp/mysql/db_mariadb_version.txt
	// 2. PostgreSQL <=17: /var/tmp/postgres/PG_VERSION
	// 3. PostgreSQL 18+: /var/tmp/postgres/[version]/docker/PG_VERSION (check common versions)
	cmd := []string{"sh", "-c", `
		# Check MySQL/MariaDB version file
		if [ -f /var/tmp/mysql/db_mariadb_version.txt ]; then
			cat /var/tmp/mysql/db_mariadb_version.txt
			exit 0
		fi

		# Check PostgreSQL version file (pre-18 location)
		if [ -f /var/tmp/postgres/PG_VERSION ]; then
			cat /var/tmp/postgres/PG_VERSION
			exit 0
		fi

		# Check PostgreSQL 18+ version files in version-specific directories
		for version_file in /var/tmp/postgres/*/docker/PG_VERSION; do
			if [ -f "$version_file" ]; then
				cat "$version_file"
				exit 0
			fi
		done

		# No version file found
		exit 0
	`}

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
	// Apple Container the second attachment aborts the probe container's boot with
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
// back volumes with a block device (Apple Container) permit either one writer or any
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
