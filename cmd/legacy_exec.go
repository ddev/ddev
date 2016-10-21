package cmd

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/spf13/cobra"
)

// LegacyExecCmd allows users to execute arbitrary bash commands within a container.
var LegacyExecCmd = &cobra.Command{
	Use:   "exec [app_name] [environment_name] '[cmd]'",
	Short: "run a command in an app container.",
	Long:  `Execs into container and runs bash commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		// The command string will be the first argument if using a stored
		// appConfig, or the third if passing in app/deploy names.
		cmdString := args[0]
		if len(args) > 2 {
			cmdString = args[2]
		}

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
			"exec",
			"-T", nameContainer,
		}

		if strings.Contains(cmdString, "drush dl") {
			// do we want to add a -y here?
			cmdString = strings.Replace(cmdString, "drush dl", "drush --root=/src/docroot dl", 1)
		}

		cmdSplit := strings.Split(cmdString, " ")
		cmdArgs = append(cmdArgs, cmdSplit...)
		err := utils.DockerCompose(cmdArgs...)
		if err != nil {
			log.Println(err)
			Failed("Could not execute command.")
		}

	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			return
		}

		if len(args) == 3 {
			return
		}

		Failed("Invalid arguments detected. Please use a command in the form of: exec [app_name] [environment_name] '[cmd]'")
	},
}

func init() {
	LegacyExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	LegacyCmd.AddCommand(LegacyExecCmd)
}
