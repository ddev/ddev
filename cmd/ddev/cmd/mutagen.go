package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// MutagenCmd is the top-level "ddev debug" command
var MutagenCmd = &cobra.Command{
	Use:   "mutagen [command]",
	Short: "Commands for mutagen status and sync, etc.",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(MutagenCmd)
}
