package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/plugins/platform"
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
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		if !platform.ComposeFileExists(app) {
			util.Failed("No docker-compose.yaml could be found for this application.")
		}

		err = app.Down()
		if err != nil {
			log.Println(err)
			util.Failed("Could not remove site: %s", app.ContainerName())
		}

		util.Success("Successfully removed the %s application.\n", app.GetName())
	},
}

func init() {
	RootCmd.AddCommand(LocalDevRMCmd)
}
