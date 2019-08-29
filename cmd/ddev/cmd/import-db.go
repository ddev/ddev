package cmd

import (
	"github.com/drud/ddev/pkg/nodeps"
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var dbSource string
var dbExtPath string
var progressOption bool

// ImportDBCmd represents the `ddev import-db` command.
var ImportDBCmd = &cobra.Command{
	Use:   "import-db",
	Short: "Pull the database of an existing project to the dev environment.",
	Long: `Pull the database of an existing project to the development environment.
The database can be provided as a SQL dump in a .sql, .sql.gz, .mysql, .mysql.gz, .zip, .tgz, or .tar.gz
format. For the zip and tar formats, the path to a .sql file within the archive
can be provided if it is not located at the top level of the archive.`,
	Example: `"ddev import-db" or "ddev import-db --src=.tarballs/junk.sql" or "ddev import-db --src=.tarballs/junk.sql.gz"`,
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

		err = app.ImportDB(dbSource, dbExtPath, progressOption)
		if err != nil {
			util.Failed("Failed to import database for %s: %v", app.GetName(), err)
		}
		util.Success("Successfully imported database for %v", app.GetName())
	},
}

func init() {
	ImportDBCmd.Flags().StringVarP(&dbSource, "src", "", "", "Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format")
	ImportDBCmd.Flags().StringVarP(&dbExtPath, "extract-path", "", "", "If provided asset is an archive, provide the path to extract within the archive.")
	ImportDBCmd.Flags().BoolVarP(&progressOption, "progress", "p", true, "Display a progress bar during import")
	app, err := ddevapp.GetActiveApp("")
	if err != nil && app != nil && !nodeps.ArrayContainsString(app.OmitContainers, "db") {
		RootCmd.AddCommand(ImportDBCmd)
	}
}
