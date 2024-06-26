package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugMigrateDatabase Migrates a database to a new type
var DebugMigrateDatabase = &cobra.Command{
	Use:   "migrate-database",
	Short: "Migrate a MySQL or MariaDB database to a different dbtype:dbversion; works only with MySQL and MariaDB, not with PostgreSQL",
	Example: `ddev debug migrate-database mysql:8.0
ddev debug migrate-database mariadb:11.4`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Can't find active project: %v", err)
		}

		newDBVersionType := args[0]
		existingDBType, err := app.GetExistingDBType()
		if err != nil {
			util.Failed("Failed to get existing DB type/version: %v", err)
		}
		if newDBVersionType == existingDBType {
			util.Success("Database type in the Docker volume is already %s", newDBVersionType)
			return
		}
		if !strings.HasPrefix(newDBVersionType, nodeps.MariaDB) && !strings.HasPrefix(newDBVersionType, nodeps.MySQL) {
			util.Failed("This command can only convert between MariaDB and MySQL")
		}
		if (nodeps.IsValidMariaDBVersion(newDBVersionType) || nodeps.IsValidMySQLVersion(newDBVersionType)) && (nodeps.IsValidMariaDBVersion(existingDBType) || nodeps.IsValidMySQLVersion(existingDBType)) {
			if !util.Confirm(fmt.Sprintf("Is it OK to attempt conversion from %s to %s?\nThis will export your database, create a snapshot,\nthen destroy your current database and import into the new database type.\nIt only migrates the 'db' database", existingDBType, newDBVersionType)) {
				util.Failed("migrate-database cancelled")
			}
			err = os.MkdirAll(app.GetConfigPath(".downloads"), 0755)
			if err != nil {
				util.Failed("Failed to create %s: %v", app.GetConfigPath(".downloads"), err)
			}

			status, _ := app.SiteStatus()
			if status != ddevapp.SiteRunning {
				err = app.Start()
				if err != nil {
					util.Failed("Failed to start %s: %v", app.Name, err)
				}
			}

			err = app.ExportDB(app.GetConfigPath(".downloads/db.sql.gz"), "gzip", "db")
			if err != nil {
				util.Failed("Failed to export-db to %s: %v", app.GetConfigPath(".downloads/db.sql.gz"), err)
			}
			err = app.Stop(true, true)
			if err != nil {
				util.Failed("Failed to stop and delete %s with snapshot: %v", app.Name, err)
			}

			typeVals := strings.Split(newDBVersionType, ":")
			app.Database.Type = typeVals[0]
			app.Database.Version = typeVals[1]
			err = app.WriteConfig()
			if err != nil {
				util.Failed("Failed to WriteConfig: %v", err)
			}
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start %s: %v", app.Name, err)
			}
			err = app.ImportDB(app.GetConfigPath(".downloads/db.sql.gz"), "", true, false, "")
			if err != nil {
				util.Failed("Failed to import-db %s: %v", app.GetConfigPath(".downloads/db.sql.gz"), err)
			}
			util.Success("Database was converted to %s", newDBVersionType)
			return
		}
		util.Failed("Invalid source database type (%s) or target database type (%s)", existingDBType, newDBVersionType)
	},
}

func init() {
	DebugCmd.AddCommand(DebugMigrateDatabase)
}
