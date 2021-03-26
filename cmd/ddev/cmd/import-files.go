package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var sourcePath string
var extPath string

// ImportFileCmd represents the `ddev import-db` command.
var ImportFileCmd = &cobra.Command{
	Use:     "import-files",
	Example: `ddev import-files --src=/path/to/files.tar.gz`,
	Short:   "Pull the uploaded files directory of an existing project to the default public upload directory of your project.",
	Long: `Pull the uploaded files directory of an existing project to the default
public upload directory of your project. The files can be provided as a
directory path or an archive in .tar, .tar.gz, .tgz, or .zip format. For the
.zip and tar formats, the path to a directory within the archive can be
provided if it is not located at the top-level of the archive. If the
destination directory exists, it will be replaced with the assets being
imported.

The destination directory can be configured in your project's config.yaml
under the upload_dir key. If no custom upload directory is defined, the app
type's default upload directory will be used.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to import files: %v", err)
		}

		var showExtPathPrompt bool
		if sourcePath == "" {
			// Ensure we prompt for extraction path if an archive is provided, while still allowing
			// non-interactive use of --src flag without providing a --extract-path flag.
			if extPath == "" {
				showExtPathPrompt = true
			}

			promptForFileSource(&sourcePath)
		}

		importPath, isArchive, err := appimport.ValidateAsset(sourcePath, "files")
		if err != nil {
			util.Failed("Failed to import files for %s: %v", app.GetName(), err)
		}

		// Ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --src flag without providing a --extract-path flag.
		if isArchive && showExtPathPrompt {
			promptForExtPath(&extPath)
		}

		if err = app.ImportFiles(importPath, extPath); err != nil {
			util.Failed("Failed to import files for %s: %v", app.GetName(), err)
		}

		util.Success("Successfully imported files for %v", app.GetName())
	},
}

const importPathPrompt = `Provide the path to the source directory or archive you wish to import.`

const importPathWarn = `Please note: if the destination directory exists, it will be replaced with the
import assets specified here.`

// promptForFileSource prompts the user for the path to the source file.
func promptForFileSource(val *string) {
	output.UserOut.Println(importPathPrompt)
	output.UserOut.Warnln(importPathWarn)

	// An empty string isn't acceptable here, keep
	// prompting until something is entered
	for {
		fmt.Print("Pull path: ")
		*val = util.GetInput("")
		if len(*val) > 0 {
			break
		}
	}
}

const extPathPrompt = `You provided an archive. Do you want to extract from a specific path in your
archive? You may leave this blank if you wish to use the full archive contents.`

// promptForExtPath prompts the user for the internal extraction path of an archive.
func promptForExtPath(val *string) {
	output.UserOut.Println(extPathPrompt)

	// An empty string is acceptable in this case, indicating
	// no particular extraction path
	fmt.Print("Archive extraction path: ")
	*val = util.GetInput("")
}

func init() {
	ImportFileCmd.Flags().StringVarP(&sourcePath, "src", "", "", "Provide the path to the source directory or tar/tar.gz/tgz/zip archive of files to import")
	ImportFileCmd.Flags().StringVarP(&extPath, "extract-path", "", "", "If provided asset is an archive, optionally provide the path to extract within the archive.")
	RootCmd.AddCommand(ImportFileCmd)
}
