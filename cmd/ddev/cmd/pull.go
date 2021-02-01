package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	// skipConfirmationArg allows a user to skip the confirmation prompt.
	skipConfirmationArg bool

	// skipDbArg allows a user to skip pulling the remote database.
	skipDbArg bool

	// skipFilesArg allows a user to skip pulling the remote file archive.
	skipFilesArg bool

	// skipImportArg allows a user to pull remote assets, but not import them into the project.
	skipImportArg bool

	// envArg allows a user to override the provider environment being pulled.
	envArg string
)

// PullCmd represents the `ddev pull` command.
var PullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull files and database using a configured provider plugin.",
	Long: `Pull files and database using a configured provider plugin.
	Running pull will connect to the configured provider and download + import the
	latest backups.`,
	Example: `ddev pull pantheon
ddev pull ddev-live
ddev pull platform`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Pull failed: %v", err)
		}
		providerName := args[0]
		p, err := app.GetProvider(providerName)
		if err != nil {
			util.Failed("No provider %s is provisioned", app.Provider)
		}
		app.ProviderInstance = p
		appPull(app, skipConfirmationArg)
	},
}

func appPull(app *ddevapp.DdevApp, skipConfirmation bool) {

	// If we're not performing the import step, we won't be deleting the existing db or files.
	if !skipConfirmation && !skipImportArg && os.Getenv("DRUD_NONINTERACTIVE") == "" {
		// Only warn the user about relevant risks.
		var message string
		if skipDbArg && skipFilesArg {
			util.Warning("Both database and files import steps skipped.")
			return
		} else if !skipDbArg && skipFilesArg {
			message = "database"
		} else if !skipFilesArg && skipDbArg {
			message = "files"
		} else {
			message = "database and files"
		}

		util.Warning("You're about to delete the current %s and replace with the results of a fresh pull.", message)
		if !util.Confirm("Would you like to continue?") {
			util.Failed("Pull cancelled")
		}
	}

	provider, err := app.GetProvider("")
	if err != nil {
		util.Failed("Failed to get provider: %v", err)
	}

	pullOpts := &ddevapp.PullOptions{
		SkipDb:      skipDbArg,
		SkipFiles:   skipFilesArg,
		SkipImport:  skipImportArg,
		Environment: envArg,
	}

	if err := app.Pull(provider, pullOpts); err != nil {
		util.Failed("Pull failed: %v", err)
	}

	util.Success("Pull succeeded.")
}

func init() {
	PullCmd.Flags().BoolVarP(&skipConfirmationArg, "skip-confirmation", "y", false, "Skip confirmation step")
	PullCmd.Flags().BoolVar(&skipDbArg, "skip-db", false, "Skip pulling database archive")
	PullCmd.Flags().BoolVar(&skipFilesArg, "skip-files", false, "Skip pulling file archive")
	PullCmd.Flags().BoolVar(&skipImportArg, "skip-import", false, "Downloads file and/or database archives, but does not import them")
	PullCmd.Flags().StringVar(&envArg, "env", "", "Overrides the default provider environment being pulled")
	RootCmd.AddCommand(PullCmd)
}
