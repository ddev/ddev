package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		app := platform.PluginMap[strings.ToLower(plugin)]

		opts := platform.AppOptions{
			Name: activeApp,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `ddev start`.")
		}

		if !platform.ComposeFileExists(app) {
			Failed("No docker-compose.yaml could be found for this application.")
		}

		err := app.Down()
		if err != nil {
			log.Println(err)
			Failed("Could not remove site: %s", app.ContainerName())
		}

		Success("Successfully removed the %s application.\n", activeApp)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevRMCmd)
}
