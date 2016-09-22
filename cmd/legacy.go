package cmd

import "github.com/spf13/cobra"

// LocalCmd represents the local command
var LegacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Local dev options",
	Long:  `Manage your local DRUD development environment with these commands.`,
}

func init() {
	RootCmd.AddCommand(LegacyCmd)
}
