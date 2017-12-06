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

// ImportDBCmd represents the `ddev import-db` command.
var ImportDBCmd = &cobra.Command{
	Use:   "import-db",
	Short: "Import the database of an existing site to the local dev environment.",
	Long: `Import the database of an existing site to the local development environment.
The database can be provided as a SQL dump in a .sql, .sql.gz, .zip, .tgz, or .tar.gz
format. For the zip and tar formats, the path to a .sql file within the archive
can be provided if it is not located at the top-level of the archive.`,
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
			util.Failed("The site is not running. The site must be running in order to import a database.")
		}

		err = app.ImportDB(dbSource, dbExtPath)
		if err != nil {
			util.Failed("Failed to import database for %s: %v", app.GetName(), err)
		}
		util.Success("Successfully imported database for %v", app.GetName())
	},
}

func init() {
	ImportDBCmd.Flags().StringVarP(&dbSource, "src", "", "", "Provide the path to a sql dump in .sql or tar/tar.gz/tgz/zip format")
	ImportDBCmd.Flags().StringVarP(&dbExtPath, "extract-path", "", "", "If provided asset is an archive, provide the path to extract within the archive.")
	RootCmd.AddCommand(ImportDBCmd)
}
