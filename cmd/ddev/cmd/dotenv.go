package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DotEnvCmd is the top-level "ddev dotenv" command
var DotEnvCmd = &cobra.Command{
	Use:   "dotenv [command]",
	Short: "Commands for managing the contents of .env files",
	Run: func(cmd *cobra.Command, _ []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(DotEnvCmd)
}
