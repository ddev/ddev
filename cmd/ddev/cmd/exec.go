package cmd

import (
	"os"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var root bool

// DdevExecCmd allows users to execute arbitrary bash commands within a container.
var DdevExecCmd = &cobra.Command{
	Use:     "exec <command>",
	Aliases: []string{"."},
	Short:   "Execute a shell command in the container for a service. Uses the web service by default.",
	Long:    `Execute a shell command in the container for a service. Uses the web service by default. To run your command in the container for another service, run "ddev exec --service <service> <cmd>". To run your command as root, run "ddev exec --root <cmd>".`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(1)
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to exec command: %v", err)
		}

		if strings.Contains(app.SiteStatus(), ddevapp.SiteNotFound) {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		if strings.Contains(app.SiteStatus(), ddevapp.SiteStopped) {
			util.Failed("Project is stopped. Run 'ddev start' to start the environment.")
		}

		app.DockerEnv()

		var out string
		if root {
			out, _, err = app.ExecRoot(serviceType, args...)
			if err != nil {
				util.Failed("Failed to execute root command %s: %v", args, err)
			}
		} else {
			out, _, err = app.Exec(serviceType, args...)
			if err != nil {
				util.Failed("Failed to execute command %s: %v", args, err)
			}
		}

		output.UserOut.Print(out)
	},
}

func init() {
	DdevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	DdevExecCmd.Flags().BoolVarP(&root, "root", "R", false, "Indicates that the command should be run as root in the target container")

	// This requires flags for exec to be specified prior to any arguments, allowing for
	// flags to be ignored by cobra for commands that are to be executed in a container.
	DdevExecCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(DdevExecCmd)
}
