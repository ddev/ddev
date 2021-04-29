package cmd

import (
	"fmt"
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// PushCmd represents the `ddev push` command.
var PushCmd = &cobra.Command{
	Use:   "push",
	Short: "push files and database using a configured provider plugin.",
	Long: `push files and database using a configured provider plugin.
	Running push will connect to the configured provider and export and upload the
	database and/or files. It is not recommended for most workflows since it is extremely dangerous to your production hosting.`,
	Example: `ddev push pantheon
ddev push platform
ddev push pantheon -y
ddev push platform --skip-files -y
ddev push acquia --skip-db -y`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		dockerutil.EnsureDdevNetwork()
	},
}

// apppush() does the work of push
func apppush(providerType string, app *ddevapp.DdevApp, skipConfirmation bool, skipImportArg bool, skipDbArg bool, skipFilesArg bool) {

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

		util.Warning("You're about to push your local %s to your upstream production\nand replace it with your local project's %s.\nThis is normally a very dangerous operation.", message, message)
		if !util.Confirm("Would you like to continue (not recommended)?") {
			util.Failed("push cancelled")
		}
	}

	provider, err := app.GetProvider(providerType)
	if err != nil {
		util.Failed("Failed to get provider: %v", err)
	}

	if err := app.Push(provider, skipDbArg, skipFilesArg); err != nil {
		util.Failed("push failed: %v", err)
	}

	util.Success("push succeeded.")
}

func init() {
	RootCmd.AddCommand(PushCmd)

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
			Short: fmt.Sprintf("push with %s", subCommandName),
			Example: fmt.Sprintf(`ddev push %s
ddev push %s -y
ddev push %s --skip-files -y`, subCommandName, subCommandName, subCommandName),
			Args: cobra.ExactArgs(0),
			Run: func(cmd *cobra.Command, args []string) {
				app, err := ddevapp.GetActiveApp("")
				if err != nil {
					util.Failed("push failed: %v", err)
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
				apppush(providerName, app, flags["skip-confirmation"], flags["skip-import"], flags["skip-db"], flags["skip-files"])
			},
		}
		PushCmd.AddCommand(subCommand)
		subCommand.Flags().BoolP("skip-confirmation", "y", false, "Skip confirmation step")
		subCommand.Flags().Bool("skip-db", false, "Skip pushing database archive")
		subCommand.Flags().Bool("skip-files", false, "Skip pushing file archive")
		subCommand.Flags().Bool("skip-import", false, "Downloads file and/or database archives, but does not import them")

	}
}
