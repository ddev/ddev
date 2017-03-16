package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/pretty"
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

		app := platform.PluginMap[strings.ToLower(plugin)]
		app.Init()

		nameContainer := fmt.Sprintf("%s-%s", app.ContainerName(), serviceType)
		if !dockerutil.IsRunning(nameContainer) {
			Failed("App not running locally. Try `ddev start`.")
		}

		app.DockerEnv()
		fmt.Println(nameContainer)
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
		fmt.Println(pretty.Prettify(cmdArgs))
		err := dockerutil.DockerCompose(cmdArgs...)
		if err != nil {
			log.Println(err)
			Failed("Could not execute command.")
		}

	},
}

func init() {
	LocalDevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Which service to send the command to. [web, db]")
	RootCmd.AddCommand(LocalDevExecCmd)
}
