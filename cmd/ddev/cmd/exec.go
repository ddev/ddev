package cmd

import (
	"log"
	"strings"

	"github.com/drud/ddev/pkg/util"
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
			util.Failed("Failed to exec command: %v", err)
		}

		labels := map[string]string{
			"com.ddev.site-name":      app.GetName(),
			"com.ddev.container-type": serviceType,
		}
		container, err := util.FindContainerByLabels(labels)
		nameContainer := util.ContainerName(container)

		if app.SiteStatus() != "running" {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		app.DockerEnv()

		cmdSplit := strings.Split(cmdString, " ")

		err = util.ContainerExec(nameContainer, cmdSplit)
		if err != nil {
			util.Failed("Failed to execute command %s: %v", cmdString, err)
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
