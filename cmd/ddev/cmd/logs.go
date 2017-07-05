package cmd

import (
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	tail      string
	follow    bool
	timestamp bool
)

// LocalDevLogsCmd ...
var LocalDevLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get the logs from your running services.",
	Long:  `Uses 'docker logs' to display stdout from the running services.`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := platform.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to retrieve logs: %v", err)
		}

		if strings.Contains(app.SiteStatus(), platform.SiteNotFound) {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		if strings.Contains(app.SiteStatus(), platform.SiteStopped) {
			util.Failed("App is stopped. Run `ddev start` to start the environment.")
		}

		err = app.Logs(serviceType, follow, timestamp, tail)
		if err != nil {
			util.Failed("Failed to retrieve logs for %s: %v", app.GetName(), err)
		}
	},
}

func init() {
	LocalDevLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	LocalDevLogsCmd.Flags().BoolVarP(&timestamp, "time", "t", false, "Add timestamps to logs")
	LocalDevLogsCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to retrieve logs from. [e.g. web, db]")
	LocalDevLogsCmd.Flags().StringVarP(&tail, "tail", "", "", "How many lines to show")
	RootCmd.AddCommand(LocalDevLogsCmd)

}
