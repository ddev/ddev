package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// pantheonEnvironmentName is the environment, dev/test/prod, defaults to "dev"
var pantheonEnvironmentName = "dev"

// pantheonConfigCommand is the the `ddev config pantheon` command
var pantheonConfigCommand *cobra.Command = &cobra.Command{
	Use:     "pantheon",
	Short:   "Create or modify a ddev project pantheon configuration in the current directory",
	Example: `"ddev config pantheon" or "ddev config pantheon --docroot=. --project-name=myproject --pantheon-environment=dev"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		providerName = nodeps.ProviderPantheon
		extraFlagsHandlingFunc = handlePantheonFlags
	},
	Run: handleConfigRun,
}

func init() {
	pantheonConfigCommand.Flags().AddFlagSet(ConfigCommand.Flags())
	pantheonConfigCommand.Flags().StringVarP(&pantheonEnvironmentName, "pantheon-environment", "", "", "Choose the environment for a Pantheon project (dev/test/prod)")

	ConfigCommand.AddCommand(pantheonConfigCommand)
}

// handlePantheonFlags is the pantheon-specific flag handler
func handlePantheonFlags(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error {
	provider, err := app.GetProvider("")
	if err != nil {
		return fmt.Errorf("failed to GetProvider: %v", err)
	}
	pantheonProvider := provider.(*ddevapp.PantheonProvider)
	err = pantheonProvider.SetSiteNameAndEnv(pantheonEnvironmentName)
	if err != nil {
		return err
	}

	return nil
}
