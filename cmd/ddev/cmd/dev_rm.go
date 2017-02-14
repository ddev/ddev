package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/local"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:   "rm [app_name] [environment_name]",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.PluginMap[strings.ToLower(plugin)]

		opts := local.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `drud legacy add`.")
		}

		if !local.ComposeFileExists(app) {
			Failed("No docker-compose yaml for this site. Try `drud legacy add`.")
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
	RootCmd.AddCommand(LocalDevRMCmd)
}
