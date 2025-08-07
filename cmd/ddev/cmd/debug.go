package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var showHidden bool

// DebugCmd is the top-level "ddev debug" command
var DebugCmd = &cobra.Command{
	Use:     "debug [command]",
	Short:   "A collection of debugging commands",
	Aliases: []string{"d", "dbg"},
	Example: `ddev debug
ddev debug mutagen sync list
ddev d mutagen sync list
ddev d capabilities
ddev debug --show-hidden`,
	Run: func(cmd *cobra.Command, _ []string) {
		if showHidden {
			// Show all commands including hidden ones
			for _, c := range cmd.Commands() {
				c.Hidden = false
			}
		}
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	DebugCmd.Flags().BoolVar(&showHidden, "show-hidden", false, "Show hidden developer/debugging commands")
	_ = DebugCmd.RegisterFlagCompletionFunc("show-hidden", configCompletionFunc([]string{"true", "false"}))
	RootCmd.AddCommand(DebugCmd)
}
