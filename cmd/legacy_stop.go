package cmd

import (
	"log"
	"path"

	"github.com/drud/bootstrap/cli/local"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// LegacyStopCmd represents the stop command
var LegacyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop an application's local services.",
	Long:  `Stop will turn off the local containers and not remove them.`,
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		app := local.LegacyApp{
			Name:        activeApp,
			Environment: activeDeploy,
		}

		err := drudutils.DockerCompose(
			"-f", path.Join(app.AbsPath(), "docker-compose.yaml"),
			"stop",
		)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {

	LegacyCmd.AddCommand(LegacyStopCmd)

}
