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

	volName := app.GetMariaDBVolumeName()
	if !dockerutil.VolumeExists(volName) {
		volName = app.GetPostgresVolumeName()
		if !dockerutil.VolumeExists(volName) {
			//output.UserOut.Printf("No database volume currently exists for project %s, one will be created when needed", app.Name)
			return "", nil
		}
	}
	_, out, err := dockerutil.RunSimpleContainer(versionconstants.GetWebImage(), "", []string{"sh", "-c", "cat /var/tmp/db_mariadb_version.txt || cat /var/tmp/PG_VERSION"}, []string{}, []string{}, []string{volName + ":/var/tmp"}, "", true, false, nil)

	if err != nil {
		util.Failed("failed to RunSimpleContainer to inspect database version/type: %v, output=%s", err, out)
	}

	dbType := ""
	dbVersion := ""
	out = strings.Trim(out, " \n\r\t")
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
