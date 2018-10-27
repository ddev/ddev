package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:   "auth [command]",
	Short: "A collection of authentication commands",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)
}
