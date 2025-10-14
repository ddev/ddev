package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var showHidden bool

// DebugCmd is the top-level "ddev utility" command
var DebugCmd = &cobra.Command{
	Use:     "utility [command]",
	Short:   "A collection of utility and debugging commands",
	Aliases: []string{"ut", "d", "dbg", "debug"},
	Example: `ddev utility
ddev utility mutagen sync list
ddev ut mutagen sync list
ddev ut test
ddev utility --show-hidden`,
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
