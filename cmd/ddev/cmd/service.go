package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ServiceCmd is the top-level "ddev service" command
var ServiceCmd = &cobra.Command{
	Use:   "service [command]",
	Short: "Add or remove, enable or disable extra services",
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(ServiceCmd)
}
