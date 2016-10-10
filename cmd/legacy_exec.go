package cmd

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/bootstrap/cli/utils"
	drudutils "github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// LegacyExecCmd allows users to execute arbitrary bash commands within a container.
var LegacyExecCmd = &cobra.Command{
	Use:   "exec '[cmd]'",
	Short: "run a command in an app container.",
	Long:  `Execs into container and runs bash commands.`,
	Run: func(cmd *cobra.Command, args []string) {

		if activeApp == "" {
			log.Fatalln("Must set app flag to dentoe which app you want to work with.")
		}

		if len(args) < 1 {
			log.Fatalln("Must pass a command as first argument.")
		}

		cmdString := args[0]

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
			"exec",
			"-T", nameContainer,
		}

		if strings.Contains(cmdString, "drush dl") {
			// do we want to add a -y here?
			cmdString = strings.Replace(cmdString, "drush dl", "drush --root=/src/docroot dl", 1)
		}

		cmdSplit := strings.Split(cmdString, " ")
		cmdArgs = append(cmdArgs, cmdSplit...)

		err := drudutils.DockerCompose(cmdArgs...)
		if err != nil {
			log.Fatalln(err)
		}

	},
}

func init() {

	LegacyExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	LegacyCmd.AddCommand(LegacyExecCmd)

}
