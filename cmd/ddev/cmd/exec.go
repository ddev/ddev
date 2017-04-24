package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/spf13/cobra"
)

// LocalDevExecCmd allows users to execute arbitrary bash commands within a container.
var LocalDevExecCmd = &cobra.Command{
	Use:   "exec '[cmd]'",
	Short: "Execute a Linux shell command in the webserver container.",
	Long:  `Execute a Linux shell command in the webserver container.`,
	Run: func(cmd *cobra.Command, args []string) {
		// The command string will be the first argument if using a stored
		// appConfig, or the third if passing in app/deploy names.
		cmdString := args[0]
		if len(args) > 2 {
			cmdString = args[2]
		}

		app, err := getActiveApp()
		if err != nil {
			log.Fatalf("Could not find an active ddev configuration, have you run 'ddev config'?: %v", err)
		}

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		app.DockerEnv()
		cmdArgs := []string{
			"-f", app.DockerComposeYAMLPath(),
			"exec",
			"-T", nameContainer,
		}

		if strings.Contains(cmdString, "drush dl") {
			// do we want to add a -y here?
			cmdString = strings.Replace(cmdString, "drush dl", "drush --root=/src/docroot dl", 1)
		}

		cmdSplit := strings.Split(cmdString, " ")
		cmdArgs = append(cmdArgs, cmdSplit...)
		err = dockerutil.DockerCompose(cmdArgs...)
		if err != nil {
			util.Failed("Could not execute command.", cmdString, err)
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("Invalid arguments detected. Please use a command in the form of `ddev exec '[cmd]'`")
		}
	},
}

func init() {
	LocalDevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	RootCmd.AddCommand(LocalDevExecCmd)
}
