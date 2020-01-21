package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var dbSource string
var dbExtPath string
var targetDB string
var progressOption bool

// ImportDBCmd represents the `ddev import-db` command.
var ImportDBCmd = &cobra.Command{
	Use:   "import-db",
	Short: "Import a sql archive into the project.",
	Long: `Import a sql archive into the project.
The database can be provided as a SQL dump in a .sql, .sql.gz, .mysql, .mysql.gz, .zip, .tgz, or .tar.gz
format. For the zip and tar formats, the path to a .sql file within the archive
can be provided if it is not located at the top level of the archive. An optional target database
can also be provided; the default is the default database named "db". 
Also note the related "ddev mysql" command`,
	Example: `ddev import-db
ddev import-db --src=.tarballs/junk.sql
ddev import-db --src=.tarballs/junk.sql.gz
ddev import-db --target-db=newdb --src=.tarballs/junk.sql.gz
ddev import-db <db.sql
gzip -dc db.sql.gz | ddev import-db`,

	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to import database: %v", err)
		}

		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to import-db: %v", app.Name, err)
			}
		}

		err = app.ImportDB(dbSource, dbExtPath, progressOption, targetDB)
		if err != nil {
			util.Failed("Failed to import database for %s: %v", app.GetName(), err)
		}
		util.Success("Successfully imported database for %v", app.GetName())
	},
}

func init() {
	ImportDBCmd.Flags().StringVarP(&dbSource, "src", "", "", "Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format")
	ImportDBCmd.Flags().StringVarP(&dbExtPath, "extract-path", "", "", "If provided asset is an archive, provide the path to extract within the archive.")
	ImportDBCmd.Flags().StringVarP(&targetDB, "target-db", "d", "db", "If provided asset is an archive, provide the path to extract within the archive.")
	ImportDBCmd.Flags().BoolVarP(&progressOption, "progress", "p", true, "Display a progress bar during import")
	RootCmd.AddCommand(ImportDBCmd)
}
