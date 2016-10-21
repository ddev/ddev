package cmd

import (
	"log"

	"github.com/drud/bootstrap/cli/local"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// LegacyRMCmd represents the stop command
var LegacyRMCmd = &cobra.Command{
	Use:   "rm [app_name] [environment_name]",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		err := app.Down()
		if err != nil {
			log.Println(err)
			Failed("Could not remove site: %s", app.ContainerName())
		}

		color.Cyan("Successfully removed the %s deploy for the %s application.\n", activeDeploy, activeApp)
	},
}

func init() {
	LegacyCmd.AddCommand(LegacyRMCmd)
}
