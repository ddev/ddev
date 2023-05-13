package ddevapp

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"regexp"
	"strings"
)

// GetExistingDBType returns type/version like mariadb:10.4 or postgres:13 or "" if no existing volume
// This has to make a docker container run so is fairly costly.
func (app *DdevApp) GetExistingDBType() (string, error) {
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.BusyboxImage, "GetExistingDBType-"+app.Name+"-"+util.RandString(6), []string{"sh", "-c", "( test -f /var/tmp/mysql/db_mariadb_version.txt && cat /var/tmp/mysql/db_mariadb_version.txt ) || ( test -f /var/tmp/postgres/PG_VERSION && cat /var/tmp/postgres/PG_VERSION) || true"}, []string{}, []string{}, []string{app.GetMariaDBVolumeName() + ":/var/tmp/mysql", app.GetPostgresVolumeName() + ":/var/tmp/postgres"}, "", true, false, map[string]string{`com.ddev.site-name`: app.GetName()}, nil)

	if err != nil {
		util.Failed("failed to RunSimpleContainer to inspect database version/type: %v, output=%s", err, out)
	}

	out = strings.Trim(out, " \n\r\t")
	// If it was empty, OK to return nothing found, even though the volume was there
	if out == "" {
		return "", nil
	}

	return dbTypeVersionFromString(out), nil
}

// dbTypeVersionFromString takes an input string and derives the info from the uses
// There are 3 possible cases here:
// 1. It has an _, meaning it's a current mysql or mariadb version. Easy to parse.
// 2. It has N+.N, meaning it's a pre-v1.19 mariadb or mysql version
// 3. It has N+, meaning it's postgres
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

	// Postgres: value is an int
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
