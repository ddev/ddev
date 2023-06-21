package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugInstrumentationCmd implements the ddev debug instrumentation command
var DebugInstrumentationCmd = &cobra.Command{
	Use:   "instrumentation [command]",
	Short: "A collection of debugging commands for instrumentation",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	DebugCmd.AddCommand(DebugInstrumentationCmd)
}
