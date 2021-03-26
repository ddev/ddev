package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var dbSource string
var dbExtPath string
var targetDB string
var noDrop bool
var progressOption bool

// ImportDBCmd represents the `ddev import-db` command.
var ImportDBCmd = &cobra.Command{
	Use:   "import-db [project]",
	Args:  cobra.RangeArgs(0, 1),
	Short: "Import a sql file into the project.",
	Long: `Import a sql file into the project.
The database dump file can be provided as a SQL dump in a .sql, .sql.gz, .mysql, .mysql.gz, .zip, .tgz, or .tar.gz
format. For the zip and tar formats, the path to a .sql file within the archive
can be provided if it is not located at the top level of the archive. An optional target database
can also be provided; the default is the default database named "db".
Also note the related "ddev mysql" command`,
	Example: `ddev import-db
ddev import-db --src=.tarballs/junk.sql
ddev import-db --src=.tarballs/junk.sql.gz
ddev import-db --target-db=newdb --src=.tarballs/junk.sql.gz
ddev import-db <db.sql
ddev import-db someproject <db.sql
gzip -dc db.sql.gz | ddev import-db`,

	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Unable to get project(s): %v", err)
		}

		app := projects[0]

		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to import-db: %v", app.Name, err)
			}
		}

		err = app.ImportDB(dbSource, dbExtPath, progressOption, noDrop, targetDB)
		if err != nil {
			util.Failed("Failed to import database %s for %s: %v", targetDB, app.GetName(), err)
		}
		util.Success("Successfully imported database '%s' for %v", targetDB, app.GetName())
		if noDrop {
			util.Success("Existing database '%s' was NOT dropped before importing", targetDB)
		} else {
			util.Success("Existing database '%s' was dropped before importing", targetDB)
		}
	},
}

func init() {
	ImportDBCmd.Flags().StringVarP(&dbSource, "src", "f", "", "Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format")
	ImportDBCmd.Flags().StringVarP(&dbExtPath, "extract-path", "", "", "If provided asset is an archive, provide the path to extract within the archive.")
	ImportDBCmd.Flags().StringVarP(&targetDB, "target-db", "d", "db", "If provided, target-db is alternate database to import into")
	ImportDBCmd.Flags().BoolVarP(&noDrop, "no-drop", "", false, "Set if you do NOT want to drop the db before importing")
	ImportDBCmd.Flags().BoolVarP(&progressOption, "progress", "p", true, "Display a progress bar during import")
	RootCmd.AddCommand(ImportDBCmd)
}
