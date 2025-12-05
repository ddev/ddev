package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:     "auth [command]",
	Short:   "A collection of authentication commands",
	Example: `ddev auth ssh`,
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)
}
