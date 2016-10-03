package cmd

import "github.com/spf13/cobra"

// LegacyCmd represents the local command
var LegacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Local dev for legacy sites.",
	Long:  `Manage your local Legacy development environment with these commands.`,
}

func init() {
	LegacyCmd.PersistentFlags().StringVarP(&activeApp, "app", "a", "", "Name of app to interact with.")
	LegacyCmd.PersistentFlags().StringVarP(&activeDeploy, "env", "e", "production", "Nme of the site environment you want to interact with.")
	RootCmd.AddCommand(LegacyCmd)
}
