package cmd

import (
	"github.com/spf13/cobra"
)

// ConfigCmd represents the config command
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set or view DRUD configurations.",
	Long:  `Set or view DRUD configurations.`,
}

func init() {
	RootCmd.AddCommand(ConfigCmd)
}
