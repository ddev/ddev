package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to remove: %v", err)
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		err = app.Down()
		if err != nil {
			util.Failed("Failed to remove %s: %s", app.GetName(), err)
		}

		util.Success("Successfully removed the %s application.\n", app.GetName())
	},
}

func init() {
	RootCmd.AddCommand(LocalDevRMCmd)
}
