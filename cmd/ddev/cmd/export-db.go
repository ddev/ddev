package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var outFileName string
var gzipOption bool
var exportTargetDB string

// ExportDBCmd is the `ddev export-db` command.
var ExportDBCmd = &cobra.Command{
	Use:   "export-db [project]",
	Short: "Dump a database to a file or to stdout",
	Long:  `Dump a database to a file or to stdout`,
	Example: `ddev export-db --file=/tmp/db.sql.gz'
ddev export-db -f /tmp/db.sql.gz
ddev export-db --gzip=false --file /tmp/db.sql
ddev export-db > /tmp/db.sql.gz
ddev export-db --gzip=false > /tmp/db.sql
ddev export-db myproject --gzip=false --file=/tmp/myproject.sql
ddev export-db someproject --gzip=false --file=/tmp/someproject.sql `,
	Args: cobra.RangeArgs(0, 1),
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
			util.Failed("ddev can't export-db until the project is started, please use ddev start.")
		}

		err = app.ExportDB(outFileName, gzipOption, exportTargetDB)
		if err != nil {
			util.Failed("Failed to export database for %s: %v", app.GetName(), err)
		}
	},
}

func init() {
	ExportDBCmd.Flags().StringVarP(&outFileName, "file", "f", "", "Provide the path to output the dump")
	ExportDBCmd.Flags().BoolVarP(&gzipOption, "gzip", "z", true, "If provided asset is an archive, provide the path to extract within the archive.")
	ExportDBCmd.Flags().StringVarP(&exportTargetDB, "target-db", "d", "db", "If provided, target-db is alternate database to export")
	RootCmd.AddCommand(ExportDBCmd)
}
