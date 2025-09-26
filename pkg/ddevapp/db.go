package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"regexp"
	"strings"
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

	return dbTypeVersionFromString(dbVersionInfo), nil
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
		for version in 18 19 20 21 22; do
			if [ -f "/var/tmp/postgres/$version/docker/PG_VERSION" ]; then
				cat "/var/tmp/postgres/$version/docker/PG_VERSION"
				exit 0
			fi
		done

		# No version file found
		exit 0
	`}

	_, out, err := dockerutil.RunSimpleContainer(
		versionconstants.UtilitiesImage,
		"GetExistingDBType-"+app.Name+"-"+util.RandString(6),
		cmd,
		[]string{}, // envVars
		[]string{}, // uid
		[]string{ // volumes
			app.GetMariaDBVolumeName() + ":/var/tmp/mysql",
			app.GetPostgresVolumeName() + ":/var/tmp/postgres",
		},
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

	return strings.Trim(out, " \n\r\t"), nil
}

// dbTypeVersionFromString takes an input string and derives the info from the uses
// There are 3 possible cases here:
// 1. It has an _, meaning it's a current MySQL or MariaDB version. Easy to parse.
// 2. It has N+.N, meaning it's a pre-v1.19 MariaDB or MySQL version
// 3. It has N+, meaning it's PostgreSQL
func dbTypeVersionFromString(in string) string {
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
