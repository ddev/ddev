package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var DebugCmd = &cobra.Command{
	Use:   "debug [command]",
	Short: "A collection of debugging commands",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(DebugCmd)
}
