package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

var (
	tail      string
	follow    bool
	timestamp bool
)

// LegacyLogsCmd ...
var LegacyLogsCmd = &cobra.Command{
	Use:   "logs [app_name] [environment_name]",
	Short: "Get the logs from your running services.",
	Long:  `Uses 'docker logs' to display stdout from the running services.`,
	Run: func(cmd *cobra.Command, args []string) {

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)

		if !utils.IsRunning(nameContainer) {
			Failed("App not running locally. Try `drud legacy add`.")
		}

		if !app.ComposeFileExists() {
			Failed("No docker-compose yaml for this site. Try `drud legacy add`.")
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

		err := utils.DockerCompose(cmdArgs...)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	LegacyLogsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the logs in real time.")
	LegacyLogsCmd.Flags().BoolVarP(&timestamp, "time", "p", false, "Add timestamps to logs")
	LegacyLogsCmd.Flags().StringVarP(&tail, "tail", "t", "", "How many lines to show")
	LegacyCmd.AddCommand(LegacyLogsCmd)

}
