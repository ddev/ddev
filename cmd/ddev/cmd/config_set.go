package cmd

import (
	"log"

	"github.com/drud/ddev/pkg/appconfig"
	"github.com/spf13/cobra"
)

// TODO: Temporary hack left in until workspace issues moved, March 2017
var (
	activeApp    string
	activeDeploy string
)

// ConfigCommand represents the `ddev config` command
var ConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Create or modify a ddev application config in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		c := appconfig.NewAppConfig()

		err := c.Config()
		if err != nil {
			log.Fatalf("There was a problem configuring your application: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(ConfigCommand)
}
