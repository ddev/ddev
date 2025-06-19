package main

import (
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	RootCmd.Execute()
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "ddev_hostname",
	Short:   "Manage hostnames in hosts file",
	Version: versionconstants.DdevVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		command := os.Args[1]
		_ = command // This is a placeholder to avoid unused variable error
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
