package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// DebugMigrateDatabase Migrates a database to a new type
var DebugMigrateDatabase = &cobra.Command{
	Use:   "migrate-database",
	Short: "Migrate a mysql or mariadb database to a different dbtype:dbversion; works only with mysql and mariadb, not with postgres",
	Example: `ddev debug migrate-database mysql:8.0
ddev debug migrate-database mariadb:10.7`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
			util.Success("database type in the docker volume is already %s", newDBVersionType)
			return
		}
		if !strings.HasPrefix(newDBVersionType, nodeps.MariaDB) && !strings.HasPrefix(newDBVersionType, nodeps.MySQL) {
			util.Failed("this command can only convert between mariadb and mysql")
		}
		if !(nodeps.IsValidMariaDBVersion(newDBVersionType) || nodeps.IsValidMySQLVersion(newDBVersionType)) && !(nodeps.IsValidMariaDBVersion(existingDBType) || nodeps.IsValidMySQLVersion(existingDBType)) {
			if !util.Confirm(fmt.Sprintf("Is it OK to attempt conversion from %s to %s?\nThis will export your database, create a snapshot,\nthen destroy your current database and import into the new database type.\nIt only migrates the 'db' database", existingDBType, newDBVersionType)) {
				util.Failed("migrate-database cancelled")
			}
			err = os.MkdirAll(app.GetConfigPath(".downloads"), 0755)
			if err != nil {
				util.Failed("failed to create %s: %v", app.GetConfigPath(".downloads"), err)
			}

			status, _ := app.SiteStatus()
			if status != ddevapp.SiteRunning {
				err = app.Start()
				if err != nil {
					util.Failed("failed to start %s: %v", app.Name, err)
				}
			}

			err = app.ExportDB(app.GetConfigPath(".downloads/db.sql.gz"), "gzip", "db")
			if err != nil {
				util.Failed("failed to export-db to %s: %v", app.GetConfigPath(".downloads/db.sql.gz"), err)
			}
			err = app.Stop(true, true)
			if err != nil {
				util.Failed("failed to stop and delete %s with snapshot: %v", app.Name, err)
			}

			typeVals := strings.Split(newDBVersionType, ":")
			app.Database.Type = typeVals[0]
			app.Database.Version = typeVals[1]
			err = app.WriteConfig()
			if err != nil {
				util.Failed("failed to WriteConfig: %v", err)
			}
			err = app.Start()
			if err != nil {
				util.Failed("failed to start %s: %v", app.Name, err)
			}
			err = app.ImportDB(app.GetConfigPath(".downloads/db.sql.gz"), "", true, false, "")
			if err != nil {
				util.Failed("failed to import-db %s: %v", app.GetConfigPath(".downloads/db.sql.gz"), err)
			}
			util.Success("database was converted to %s", newDBVersionType)
			return
		}
		util.Failed("Invalid target source database type (%s) or target database type (%s)", existingDBType, newDBVersionType)
	},
}

func init() {
	DebugCmd.AddCommand(DebugMigrateDatabase)
}
