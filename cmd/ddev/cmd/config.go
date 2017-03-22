package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		appRoot, err := filepath.Abs(filepath.Dir(os.Args[0]))
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

		err = c.Config(nil)
		if err != nil {
			log.Fatalf("There was a problem configuring your application: %v\n", err)
		}

		err = c.Write()
		if err != nil {
			log.Fatalf("Could not write ddev config file: %v\n", err)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// We need to override the PersistentPrerun which checks for a config.yaml in this instance,
		// since we're actually generating the config here.
	},
}

func init() {
	RootCmd.AddCommand(ConfigCommand)
}
