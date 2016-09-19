package cmd

import "github.com/spf13/cobra"

// LocalCmd represents the local command
var LegacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Local dev options",
	Long:  `Manage your local DRUD development environment with these commands.`,
}

func init() {
	//RootCmd.AddCommand(localCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// localCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	RootCmd.AddCommand(LegacyCmd)
}
