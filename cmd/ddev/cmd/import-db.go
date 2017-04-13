package cmd

import (
	"log"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

var dbSource string

// ImportDBCmd represents the `ddev import-db` command.
var ImportDBCmd = &cobra.Command{
	Use:   "import-db",
	Short: "Import the database of an existing site to the local dev environment.",
	Long:  "Import the database of an existing site to the local development environment. The database can be provided as a SQL dump in a .sql or .tar.gz format. For the .tar.gz format, a SQL dump in .sql format must be present at the root of the archive.",
	PreRun: func(cmd *cobra.Command, args []string) {

		client, err := platform.GetDockerClient()
		if err != nil {
			log.Fatal(err)
		}

		err = EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		err = app.ImportDB(dbSource)
		if err != nil {
			Failed("Failed to import database for %s: %s", app.GetName(), err)
		}
		Success("Successfully imported database for %s", app.GetName())
	},
}

func init() {
	ImportDBCmd.Flags().StringVarP(&dbSource, "src", "", "", "Provide the path to a sql dump in .sql or .tar.gz format")
	RootCmd.AddCommand(ImportDBCmd)
}
