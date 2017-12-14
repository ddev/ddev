package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var skipConfirmation bool

// PullCmd represents the `ddev pull` command.
var PullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Import files and database using a configured provider plugin.",
	Long: `Import files and database using a configured provider plugin.
	Running pull will connect to the configured provider and download + import the
	latest backups.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		appImport(skipConfirmation)
	},
}

func appImport(skipConfirmation bool) {
	app, err := ddevapp.GetActiveApp("")

	if err != nil {
		util.Failed("appImport failed: %v", err)
	}

	if !skipConfirmation {
		// Unfortunately we cannot use util.Warning here as it automatically adds a newline, which is awkward when dealing with prompts.
		d := color.New(color.FgYellow)
		_, err = d.Printf("You're about to delete the current database and files and replace with a fresh import. Would you like to continue (y/N): ")
		util.CheckErr(err)
		if !util.AskForConfirmation() {
			util.Warning("Import cancelled.")
			os.Exit(2)
		}
	}

	err = app.Import()
	if err != nil {
		util.Failed("Could not perform import: %v", err)
	}

	util.Success("Successfully Imported.")
	util.Success("Your application can be reached at: %s", app.GetURL())
}

func init() {
	PullCmd.Flags().BoolVarP(&skipConfirmation, "skip-confirmation", "y", false, "Skip confirmation step.")
	RootCmd.AddCommand(PullCmd)
}
