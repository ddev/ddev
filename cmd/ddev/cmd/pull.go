package cmd

import (
	"fmt"
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// PullCmd represents the `ddev pull` command.
var PullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull files and database using a configured provider plugin.",
	Long: `Pull files and database using a configured provider plugin.
	Running pull will connect to the configured provider and download + import the
	database and files.`,
	Example: `ddev pull pantheon
ddev pull platform
ddev pull pantheon -y
ddev pull platform --skip-files -y`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
}

// appPull() does the work of pull
func appPull(providerType string, app *ddevapp.DdevApp, skipConfirmation bool, skipImportArg bool, skipDbArg bool, skipFilesArg bool) {

	// If we're not performing the import step, we won't be deleting the existing db or files.
	if !skipConfirmation && !skipImportArg && os.Getenv("DDEV_NONINTERACTIVE") == "" {
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

	provider, err := app.GetProvider(providerType)
	if err != nil {
		util.Failed("Failed to get provider: %v", err)
	}

	if err := app.Pull(provider, skipDbArg, skipFilesArg, skipImportArg); err != nil {
		util.Failed("Pull failed: %v", err)
	}

	util.Success("Pull succeeded.")
}

func init() {
	RootCmd.AddCommand(PullCmd)

	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return
	}
	pList, err := app.GetValidProviders()
	if err != nil {
		return
	}
	for _, p := range pList {
		subCommandName := p
		subCommand := &cobra.Command{
			Use:   subCommandName,
			Short: fmt.Sprintf("Pull with %s", subCommandName),
			Example: fmt.Sprintf(`ddev pull %s
ddev pull %s -y
ddev pull %s --skip-files -y`, subCommandName, subCommandName, subCommandName),
			Args: cobra.ExactArgs(0),
			Run: func(cmd *cobra.Command, args []string) {
				app, err := ddevapp.GetActiveApp("")
				if err != nil {
					util.Failed("Pull failed: %v", err)
				}
				providerName := subCommandName
				p, err := app.GetProvider(subCommandName)
				if err != nil {
					util.Failed("No provider `%s' is provisioned in %s: %v", providerName, app.GetConfigPath("providers"), err)
				}
				app.ProviderInstance = p

				flags := map[string]bool{"skip-confirmation": false, "skip-db": false, "skip-files": false, "skip-import": false}
				for f := range flags {
					flags[f], err = cmd.Flags().GetBool(f)
					if err != nil {
						util.Failed("Failed to get flag %s: %v", f, err)
					}
				}

				appPull(providerName, app, flags["skip-confirmation"], flags["skip-import"], flags["skip-db"], flags["skip-files"])
			},
		}
		PullCmd.AddCommand(subCommand)
		subCommand.Flags().BoolP("skip-confirmation", "y", false, "Skip confirmation step")
		subCommand.Flags().Bool("skip-db", false, "Skip pulling database archive")
		subCommand.Flags().Bool("skip-files", false, "Skip pulling file archive")
		subCommand.Flags().Bool("skip-import", false, "Downloads file and/or database archives, but does not import them")

	}
}
