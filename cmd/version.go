package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print DRUD version",
	Long:  `Display the version of this DRUD binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		fmt.Println(cliVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
