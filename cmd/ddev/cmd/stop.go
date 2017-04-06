package cmd

import (
	log "github.com/Sirupsen/logrus"

	"github.com/spf13/cobra"
)

// LocalDevStopCmd represents the stop command
var LocalDevStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop an application's local services.",
	Long:  `Stop will turn off the local containers and not remove them.`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		err = app.Stop()
		if err != nil {
			log.Println(err)
			Failed("Failed to stop containers for %s. Run `ddev list` to ensure your site exists.", app.ContainerName())
		}

		Success("Application has been stopped.")
	},
}

func init() {
	RootCmd.AddCommand(LocalDevStopCmd)
}
