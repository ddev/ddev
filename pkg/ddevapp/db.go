package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/versionconstants"
	"strconv"
	"strings"
)

// GetExistingDBType returns type/version like mariadb:10.4 or postgres:13 or "" if no existing volume
// This has to make a docker container run so is fairly costly.
func (app *DdevApp) GetExistingDBType() (string, error) {
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.GetWebImage(), "", []string{"sh", "-c", "( test -f /var/tmp/mysql/db_mariadb_version.txt && cat /var/tmp/mysql/db_mariadb_version.txt ) || ( test -f /var/tmp/postgres/PG_VERSION && cat /var/tmp/postgres/PG_VERSION) || true"}, []string{}, []string{}, []string{app.GetMariaDBVolumeName() + ":/var/tmp/mysql", app.GetPostgresVolumeName() + ":/var/tmp/postgres"}, "", true, false, nil)

	if err != nil {
		util.Failed("failed to RunSimpleContainer to inspect database version/type: %v, output=%s", err, out)
	}

	dbType := ""
	dbVersion := ""
	out = strings.Trim(out, " \n\r\t")
	// If it was empty, OK to return nothing found, even though the volume was there
	if out == "" {
		return "", nil
	}
	if _, err := strconv.Atoi(out); err == nil {
		dbType = nodeps.Postgres
		dbVersion = out
	} else {
		res := strings.Split(out, "_")
		if len(res) != 2 {
			return "", fmt.Errorf("could not split version string %s", out)
		}
		dbType = res[0]
		dbVersion = res[1]
	}

	return dbType + ":" + dbVersion, nil
}
