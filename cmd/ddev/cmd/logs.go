package cmd

import (
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
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to retrieve logs: %v", err)
		}

		if app.SiteStatus() != "running" {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		if !platform.ComposeFileExists(app) {
			util.Failed("No docker-compose yaml for this site. Try `ddev start`.")
		}

		cmdArgs := []string{"logs"}

		if tail != "" {
			cmdArgs = append(cmdArgs, "--tail="+tail)
		}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		if timestamp {
			cmdArgs = append(cmdArgs, "-t")
		}
		cmdArgs = append(cmdArgs, serviceType)

		app.DockerEnv()
		err = util.ComposeCmd([]string{app.DockerComposeYAMLPath()}, cmdArgs...)
		if err != nil {
			util.Failed("Failed to retrieve logs for %s: %v", app.GetName(), err)
		}
	},
}

func init() {
	LocalDevLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	LocalDevLogsCmd.Flags().BoolVarP(&timestamp, "time", "s", false, "Add timestamps to logs")
	LocalDevLogsCmd.Flags().StringVarP(&tail, "tail", "t", "", "How many lines to show")
	RootCmd.AddCommand(LocalDevLogsCmd)

}
