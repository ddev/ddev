package cmd

import (
	"fmt"
	"path"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
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
			util.Failed("Failed to retrieve logs for %s: %s", app.GetName(), err)
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)

		if !dockerutil.IsRunning(nameContainer) {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		if !platform.ComposeFileExists(app) {
			util.Failed("No docker-compose yaml for this site. Try `ddev start`.")
		}

		cmdArgs := []string{
			"-f", path.Join(app.DockerComposeYAMLPath()),
			"logs",
		}

		if tail != "" {
			cmdArgs = append(cmdArgs, "--tail="+tail)
		}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		if timestamp {
			cmdArgs = append(cmdArgs, "-t")
		}
		cmdArgs = append(cmdArgs, nameContainer)

		app.DockerEnv()
		err = dockerutil.DockerCompose(cmdArgs...)
		if err != nil {
			util.Failed("Failed to retrieve logs for %s: %s", app.GetName(), err)
		}
	},
}

func init() {
	LocalDevLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	LocalDevLogsCmd.Flags().BoolVarP(&timestamp, "time", "s", false, "Add timestamps to logs")
	LocalDevLogsCmd.Flags().StringVarP(&tail, "tail", "t", "", "How many lines to show")
	RootCmd.AddCommand(LocalDevLogsCmd)

}
