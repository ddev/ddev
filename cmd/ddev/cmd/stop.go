package cmd

import (
	"github.com/drud/ddev/pkg/util"
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
			util.Failed("Failed to stop: %v", err)
		}

		err = app.Stop()
		if err != nil {
			util.Failed("Failed to stop containers for %s: %v", app.ContainerName(), err)
		}

		util.Success("Application has been stopped.")
	},
}

func init() {
	RootCmd.AddCommand(LocalDevStopCmd)
}
