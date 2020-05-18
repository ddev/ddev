package cmd

import (
	"github.com/drud/ddev/pkg/nodeps"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

var ddevListOrgName string

// ddevliveConfigCommand is the the `ddev config ddevlive` command
var ddevliveConfigCommand *cobra.Command = &cobra.Command{
	Use:     "ddev-live",
	Short:   "Create or modify a DDEV-Live configuration in the current directory",
	Example: `"ddev config ddev-live" or "ddev config ddev-live --docroot=. --project-name=myproject"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		providerName = nodeps.ProviderDdevLive
		extraFlagsHandlingFunc = handleDdevLiveFlags
	},
	Run: handleConfigRun,
}

func init() {
	ddevliveConfigCommand.Flags().AddFlagSet(ConfigCommand.Flags())
	ddevliveConfigCommand.Flags().StringVarP(&ddevListOrgName, "org", "", "", "provide the DDEV-Live org name")
	ConfigCommand.AddCommand(ddevliveConfigCommand)
}

// handleDdevLiveFlags is the ddevlive-specific flag handler
func handleDdevLiveFlags(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error {
	//provider, err := app.GetProvider()
	//if err != nil {
	//	return fmt.Errorf("failed to GetProvider: %v", err)
	//}
	//ddevliveProvider := provider.(*ddevapp.DdevLiveProvider)
	//if err != nil {
	//	return err
	//}

	return nil
}
