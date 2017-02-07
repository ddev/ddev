package cmd

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"

	"github.com/drud/ddev/pkg/local"

	"github.com/spf13/cobra"
)

// LocalDevStopCmd represents the stop command
var LocalDevStopCmd = &cobra.Command{
	Use:   "stop [app_name] [environment_name]",
	Short: "Stop an application's local services.",
	Long:  `Stop will turn off the local containers and not remove them.`,
	Run: func(cmd *cobra.Command, args []string) {
		app := local.PluginMap[strings.ToLower(plugin)]

		opts := local.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
		}
		app.SetOpts(opts)

		err := app.Stop()
		if err != nil {
			log.Println(err)
			Failed("Failed to stop containers for %s. Run 'drud legacy list' to ensure your site exists.", app.ContainerName())
		}

		color.Cyan("Application has been stopped.")
	},
}

func init() {
	RootCmd.AddCommand(LocalDevStopCmd)
}
