package cmd

import (
	"log"
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		appRoot, err := os.Getwd()
		if err != nil {
			log.Fatalf("Could not determine current working directory: %v\n", err)
		}

		c, err := ddevapp.NewConfig(appRoot)
		if err != nil {
			// If there is an error reading the config and the file exists, we're not sure
			// how to proceed.
			if c.ConfigExists() {
				log.Fatalf("Could not read config: %v", err)
			}
		}

		err = c.Config()
		if err != nil {
			log.Fatalf("There was a problem configuring your application: %v\n", err)
		}

		err = c.Write()
		if err != nil {
			log.Fatalf("Could not write ddev config file: %v\n", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(ConfigCommand)
}
