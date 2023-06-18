package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugCmd is the top-level "ddev debug" command
var DebugCmd = &cobra.Command{
	Use:     "debug [command]",
	Short:   "A collection of debugging commands",
	Aliases: []string{"d", "dbg"},
	Example: `ddev debug
ddev debug mutagen sync list
ddev d mutagen sync list
ddev d capabilities`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(DebugCmd)
}
