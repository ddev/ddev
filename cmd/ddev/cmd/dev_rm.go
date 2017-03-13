package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
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
		app := platform.PluginMap[strings.ToLower(plugin)]

		opts := platform.AppOptions{
			Name: activeApp,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `dev add`.")
		}

		if !platform.ComposeFileExists(app) {
			Failed("No docker-compose yaml for this site. Try `dev add`.")
		}

		err := app.Down()
		if err != nil {
			log.Println(err)
			Failed("Could not remove site: %s", app.ContainerName())
		}

		color.Cyan("Successfully removed the %s application.\n", activeApp)
	},
}

func init() {
	RootCmd.AddCommand(LocalDevRMCmd)
}
