package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		appRoot, err := os.Getwd()
		if err != nil {
			util.Failed("Could not determine current working directory: %v\n", err)
		}

		c, err := ddevapp.NewConfig(appRoot)
		if err != nil {
			// If there is an error reading the config and the file exists, we're not sure
			// how to proceed.
			if c.ConfigExists() {
				util.Failed("Could not read config: %v", err)
			}
		}

		err = c.Config()
		if err != nil {
			util.Failed("There was a problem configuring your application: %v\n", err)
		}

		err = c.Write()
		if err != nil {
			util.Failed("Could not write ddev config file: %v\n", err)
		}
		util.Success("Initial configuration file written successfully. See the configuration file for additional configuration options.")
	},
}

func init() {
	RootCmd.AddCommand(ConfigCommand)
}
