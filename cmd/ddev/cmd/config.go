package cmd

import (
	"log"
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config [provider]",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {

		appRoot, err := os.Getwd()
		if err != nil {
			util.Failed("Could not determine current working directory: %v\n", err)

		}

		provider := ddevapp.DefaultProviderName

		if len(args) > 1 {
			log.Fatal("Invalid argument detected. Please use 'ddev config' or 'ddev config [provider]' to configure a site.")
		}

		if len(args) == 1 {
			provider = args[0]
		}

		c, err := ddevapp.NewConfig(appRoot, provider)
		if err != nil {
			// If there is an error reading the config and the file exists, we're not sure
			// how to proceed.
			if c.ConfigExists() {
				util.Failed("Could not read config: %v", err)
			}
		}

		// Set the provider value after load so we can ensure we use the passed in provider value
		// for this configuration.
		c.Provider = provider

		err = c.PromptForConfig()
		if err != nil {
			util.Failed("There was a problem configuring your application: %v\n", err)
		}

		err = c.Write()
		if err != nil {
			util.Failed("Could not write ddev config file: %v\n", err)

		}

		// If a provider is specified, prompt about whether to do an import after config.
		switch provider {
		case ddevapp.DefaultProviderName:
			util.Success("Configuration complete. You may now run 'ddev start'.")
		default:
			util.Success("Configuration complete. You may now run 'ddev start' or 'ddev pull'")
		}
	},
}

func init() {
	RootCmd.AddCommand(ConfigCommand)
}
