package cmd

import "github.com/spf13/cobra"

var DebugCmd = &cobra.Command{
	Use:   "debug [command]",
	Short: "A collection of debugging commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func init() {
	RootCmd.AddCommand(DebugCmd)
}
