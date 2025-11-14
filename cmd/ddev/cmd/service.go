package cmd

import (
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ServiceCmd is the top-level "ddev service" command
var ServiceCmd = &cobra.Command{
	Use:    "service",
	Short:  "The service commands have been removed",
	Hidden: true,
	Run: func(_ *cobra.Command, _ []string) {
		util.Failed("The 'ddev service' command has been removed. Please use 'ddev add-on' commands instead.\n\nSee https://docs.ddev.com/en/stable/users/usage/commands/#add-on for more information.")
	},
}

func init() {
	RootCmd.AddCommand(ServiceCmd)
}
