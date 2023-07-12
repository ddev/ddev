package cmd

import (
	"fmt"

	"github.com/ddev/ddev/pkg/appimport"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// NewImportFileCmd initialized and return the `ddev import-db` command.
func NewImportFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-files",
		Short: "Pull the uploaded files directory of an existing project to the default public upload directory of your project",
		Long: heredoc.Doc(`
			Pull the uploaded files directory of an existing project to the default
			public upload directory of your project. The files can be provided as a
			directory path or an archive in .tar, .tar.gz, .tar.xz, .tar.bz2, .tgz, or .zip format. For the
			.zip and tar formats, the path to a directory within the archive can be
			provided if it is not located at the top-level of the archive. If the
			destination directory exists, it will be replaced with the assets being
			imported.

			The destination directories can be configured in your project's config.yaml
			under the upload_dirs key. If no custom upload directory is defined, the app
			type's default upload directory will be used.
		`),
		Example: heredoc.DocI2S(`
			ddev import-files --source=/path/to/files.tar.gz
			ddev import-files --source=/path/to/dir
			ddev import-files --source=/path/to/files.tar.xz
			ddev import-files --source=/path/to/files.tar.bz2
			ddev import-files --source=.tarballs/files.tar.xz --target=../private
			ddev import-files --source=.tarballs/files.tar.gz --target=sites/default/files
		`),
		PreRun: func(cmd *cobra.Command, args []string) {
			dockerutil.EnsureDdevNetwork()
		},
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := ddevapp.GetActiveApp("")
			if err != nil {
				return fmt.Errorf("unable to get project: %v", err)
			}

			target, err := cmd.Flags().GetString("target")
			if err != nil {
				return err
			}

			sourcePath, err := cmd.Flags().GetString("source")
			if err != nil {
				return err
			}

			if !cmd.Flags().Lookup("source").Changed && cmd.Flags().Lookup("src").Changed {
				sourcePath, err = cmd.Flags().GetString("src")
				if err != nil {
					return err
				}
			}

			extractPath, err := cmd.Flags().GetString("extract-path")
			if err != nil {
				return err
			}

			return importFilesRun(app, target, sourcePath, extractPath)
		},
	}

	cmd.Flags().StringP("target", "t", "", "Target upload dir, defaults to the first upload dir")
	cmd.Flags().StringP("source", "s", "", "Path to the source directory or source archive in `.tar`, `.tar.gz`, `.tar.bz2`, `.tar.xz`, `.tgz`, or `.zip` format")
	cmd.Flags().String("extract-path", "", "Path to extract within the archive")

	// Backward compatibility
	cmd.Flags().String("src", "", cmd.Flags().Lookup("source").Usage)
	_ = cmd.Flags().MarkDeprecated("src", "please use --source or -s instead")

	return cmd
}

func importFilesRun(app *ddevapp.DdevApp, uploadDir, sourcePath, extractPath string) error {
	var showExtPathPrompt bool
	if sourcePath == "" {
		// Ensure we prompt for extraction path if an archive is provided, while still allowing
		// non-interactive use of --source flag without providing a --extract-path flag.
		if extractPath == "" {
			showExtPathPrompt = true
		}

		promptForFileSource(&sourcePath)
	}

	importPath, isArchive, err := appimport.ValidateAsset(sourcePath, "files")
	if err != nil {
		return fmt.Errorf("failed to import files for %s: %v", app.GetName(), err)
	}

	// Ensure we prompt for extraction path if an archive is provided, while still allowing
	// non-interactive use of --source flag without providing a --extract-path flag.
	if isArchive && showExtPathPrompt {
		promptForExtractPath(&extractPath)
	}

	if err = app.ImportFiles(uploadDir, importPath, extractPath); err != nil {
		return fmt.Errorf("failed to import files for %s: %v", app.GetName(), err)
	}

	util.Success("Successfully imported files for %v", app.GetName())

	return nil
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
		fmt.Print("Path to file(s): ")
		*val = util.GetInput("")
		if len(*val) > 0 {
			break
		}
	}
}

const extPathPrompt = `You provided an archive. Do you want to extract from a specific path in your
archive? You may leave this blank if you wish to use the full archive contents.`

// promptForExtractPath prompts the user for the internal extraction path of an archive.
func promptForExtractPath(val *string) {
	output.UserOut.Println(extPathPrompt)

	// An empty string is acceptable in this case, indicating
	// no particular extraction path
	fmt.Print("Archive extraction path: ")
	*val = util.GetInput("")
}

func init() {
	RootCmd.AddCommand(NewImportFileCmd())
}
