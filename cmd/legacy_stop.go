package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"

	"github.com/drud/bootstrap/cli/local"

	"github.com/spf13/cobra"
)

// LegacyStopCmd represents the stop command
var LegacyStopCmd = &cobra.Command{
	Use:   "stop [app_name] [environment_name]",
	Short: "Stop an application's local services.",
	Long:  `Stop will turn off the local containers and not remove them.`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.NewLegacyApp(activeApp, activeDeploy)

		err := app.Stop()
		if err != nil {
			log.Println(err)
			Failed("Failed to stop containers for %s. Run 'drud legacy list' to ensure your site exists.", app.ContainerName())
		}

		color.Cyan("Application has been stopped.")
	},
}

func init() {

	LegacyCmd.AddCommand(LegacyStopCmd)

}
