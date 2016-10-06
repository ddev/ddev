package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

var (
	tail      string
	follow    bool
	timestamp bool
)

// LegacyLogsCmd ...
var LegacyLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get the logs from your running services.",
	Long:  `Uses 'docker logs' to display stdout from the running services.`,
	Run: func(cmd *cobra.Command, args []string) {

		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)

		if !utils.IsRunning(nameContainer) {
			log.Fatal("App not running locally. Try `drud legacy add`.")
		}

		if !app.ComposeFileExists() {
			log.Fatalln("No docker-compose yaml for this site. Try `drud legacy add`.")
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

		err := drudutils.DockerCompose(cmdArgs...)
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
