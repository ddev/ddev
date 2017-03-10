package cmd

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
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
	Use:   "logs [app_name] [environment_name]",
	Short: "Get the logs from your running services.",
	Long:  `Uses 'docker logs' to display stdout from the running services.`,
	Run: func(cmd *cobra.Command, args []string) {

		app := platform.PluginMap[strings.ToLower(plugin)]
		opts := platform.AppOptions{
			Name:        activeApp,
			Environment: activeDeploy,
		}
		app.SetOpts(opts)

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)

		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `ddev add`.")
		}

		if !platform.ComposeFileExists(app) {
			Failed("No docker-compose yaml for this site. Try `ddev add`.")
		}

		cmdArgs := []string{
			"-f", path.Join(app.AbsPath(), "docker-compose.yaml"),
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

		err := dockerutil.DockerCompose(cmdArgs...)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	LocalDevLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	LocalDevLogsCmd.Flags().BoolVarP(&timestamp, "time", "s", false, "Add timestamps to logs")
	LocalDevLogsCmd.Flags().StringVarP(&tail, "tail", "t", "", "How many lines to show")
	RootCmd.AddCommand(LocalDevLogsCmd)

}
