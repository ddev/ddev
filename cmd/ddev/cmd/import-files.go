package cmd

import (
	"log"
	"os"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var fileSource string
var fileExtPath string

// ImportFileCmd represents the `ddev import-db` command.
var ImportFileCmd = &cobra.Command{
	Use:   "import-files",
	Short: "Import the uploaded files directory of an existing site to the default public upload directory of your application.",
	Long: `Import the uploaded files directory of an existing site to the default public
upload directory of your application. The files can be provided as a directory
path or an archive in .tar, .tar.gz, .tgz, or .zip format. For the .zip and tar formats,
the path to a directory within the archive can be provided if it is not located at the
top-level of the archive. If the destination directory exists, it will be replaced with
the assets being imported.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}
		client := dockerutil.GetDockerClient()
		err := dockerutil.EnsureNetwork(client, netName)
		if err != nil {
			log.Fatal(err)
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := platform.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to import files: %v", err)
		}

		err = app.ImportFiles(fileSource, fileExtPath)
		if err != nil {
			util.Failed("Failed to import files for %s: %v", app.GetName(), err)
		}
		util.Success("Successfully imported files for %v", app.GetName())
	},
}

func init() {
	ImportFileCmd.Flags().StringVarP(&fileSource, "src", "", "", "Provide the path to a directory or tar/tar.gz/tgz/zip archive of files to import")
	ImportFileCmd.Flags().StringVarP(&fileExtPath, "extract-path", "", "", "If provided asset is an archive, provide the path to extract within the archive.")
	RootCmd.AddCommand(ImportFileCmd)
}
