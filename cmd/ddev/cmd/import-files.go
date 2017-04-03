package cmd

import (
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

var fileSource string

// ImportFileCmd represents the `ddev import-db` command.
var ImportFileCmd = &cobra.Command{
	Use:   "import-files",
	Short: "Import the uploaded files directory of an existing site to the local dev environment.",
	Long:  "Import the uploaded files directory of an existing site to the local development environment. The files can be provided as a directory path or an archive in .tar.gz format. For the .tar.gz format, the contents of the uploaded files directory must be present at the root of the archive.",
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
		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()
		err := app.ImportFiles(fileSource)
		if err != nil {
			Failed("Unable to successfully import files for %s: %s", app.GetName(), err)
		}
		Success("Successfully imported files for %s", app.GetName())
	},
}

func init() {
	ImportFileCmd.Flags().StringVarP(&fileSource, "src", "", "", "Provide the path to a directory or .tar.gz archive of files to import")
	RootCmd.AddCommand(ImportFileCmd)
}
