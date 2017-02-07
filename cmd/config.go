package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ConfigCmd represents the config command
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set or view DRUD configurations.",
	Long:  `Set or view DRUD configurations.`,
}

// isFlagPresent determines if a flag has been provided for set/unset
func isFlagPresent(cmd *cobra.Command) bool {
	args := os.Args

	if strings.HasPrefix(args[3], "--") {
		return true
	}

	return false
}

func init() {
	RootCmd.AddCommand(ConfigCmd)
}
