package cmd

import (
	"log"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

var fileSource string

// ImportFileCmd represents the `ddev import-db` command.
var ImportFileCmd = &cobra.Command{
	Use:   "import-files",
	Short: "Import the uploaded files directory of an existing site to the default public upload directory of your application.",
	Long:  "Import the uploaded files directory of an existing site to the default public upload directory of your application. The files can be provided as a directory path or an archive in .tar.gz format. For the .tar.gz format, the contents of the uploaded files directory must be present at the root of the archive. If the destination directory exists, it will be removed in favor of the assets being imported.",
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

		err = app.ImportFiles(fileSource)
		if err != nil {
			Failed("Failed to import files for %s: %s", app.GetName(), err)
		}
		Success("Successfully imported files for %s", app.GetName())
	},
}

func init() {
	ImportFileCmd.Flags().StringVarP(&fileSource, "src", "", "", "Provide the path to a directory or .tar.gz archive of files to import")
	RootCmd.AddCommand(ImportFileCmd)
}
