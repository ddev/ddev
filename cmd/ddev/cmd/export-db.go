package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var outFileName string
var gzipOption bool

// ImportDBCmd represents the `ddev import-db` command.
var ExportDBCmd = &cobra.Command{
	Use:   "export-db",
	Short: "Dump a database",
	Long:  `Dump a database`,
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
			util.Failed("Failed to find project from which to export database: %v", err)
		}

		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to export-db: %v", app.Name, err)
			}
		}

		err = app.ExportDB(outFileName, gzipOption)
		if err != nil {
			util.Failed("Failed to export database for %s: %v", app.GetName(), err)
		}
	},
}

func init() {
	ExportDBCmd.Flags().StringVarP(&outFileName, "file", "f", "", "Provide the path to output the dump")
	ExportDBCmd.Flags().BoolVarP(&gzipOption, "gzip", "z", true, "If provided asset is an archive, provide the path to extract within the archive.")
	RootCmd.AddCommand(ExportDBCmd)
}
