package cmd

import (
	"log"
	"path"

	"github.com/drud/bootstrap/cli/local"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// localStopCmd represents the stop command
var LegacyRMCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove an application's local services.",
	Long:  `Remove will delete the local service containers from this machine..`,
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
			"down",
		)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {
	LegacyCmd.AddCommand(LegacyRMCmd)

}
