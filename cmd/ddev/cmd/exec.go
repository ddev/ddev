package cmd

import (
	"os"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// execDirArg allows a configurable container execution directory
var execDirArg string

// DdevExecCmd allows users to execute arbitrary sh commands within a container.
var DdevExecCmd = &cobra.Command{
	Use:     "exec <command>",
	Aliases: []string{"."},
	Short:   "Execute a shell command in the container for a service. Uses the web service by default.",
	Long:    `Execute a shell command in the container for a service. Uses the web service by default. To run your command in the container for another service, run "ddev exec --service <service> <cmd>"`,
	Example: "ddev exec ls /var/www/html\nddev exec --service db\nddev exec -s db\nddev exec -s solr (assuming an add-on service named 'solr')",
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

		if strings.Contains(app.SiteStatus(), ddevapp.SiteStopped) {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		if strings.Contains(app.SiteStatus(), ddevapp.SitePaused) {
			util.Failed("Project is paused. Run 'ddev start' to start it.")
		}

		app.DockerEnv()

		out, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: serviceType,
			Dir:     execDirArg,
			Cmd:     strings.Join(args, " "),
			Tty:     true,
		})

		if err != nil {
			util.Failed("Failed to execute command %s: %v", strings.Join(args, " "), err)
		}
		output.UserOut.Print(out)
	},
}

func init() {
	DdevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	DdevExecCmd.Flags().StringVarP(&execDirArg, "dir", "d", "", "Defines the execution directory within the container")
	// This requires flags for exec to be specified prior to any arguments, allowing for
	// flags to be ignored by cobra for commands that are to be executed in a container.
	DdevExecCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(DdevExecCmd)
}
