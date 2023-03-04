package cmd

import (
	"os"

	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// execDirArg allows a configurable container execution directory
var execDirArg string

// DdevExecCmd allows users to execute arbitrary sh commands within a container.
var DdevExecCmd = &cobra.Command{
	Use:     "exec <command>",
	Aliases: []string{"."},
	Short:   "Execute a shell command in the container for a service. Uses the web service by default.",
	Long:    `Execute a shell command in the container for a service. Uses the web service by default. To run your command in the container for another service, run "ddev exec --service <service> <cmd>". If you want to use raw, uninterpreted command inside container use --raw as in example.`,
	Example: `ddev exec ls /var/www/html
ddev exec --service db
ddev exec -s db
ddev exec -s solr (assuming an add-on service named 'solr')
ddev exec --raw -- ls -lR`,
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

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		app.DockerEnv()

		opts := &ddevapp.ExecOpts{
			Service: serviceType,
			Dir:     execDirArg,
			Cmd:     strings.Join(args, " "),
			Tty:     true,
		}

		// If they've chosen raw, use the actual passed values
		if cmd.Flag("raw").Changed {
			if useRaw, _ := cmd.Flags().GetBool("raw"); useRaw {
				opts.RawCmd = args
			}
		}

		_, _, err = app.Exec(opts)

		if err != nil {
			util.Failed("Failed to execute command %s: %v", strings.Join(args, " "), err)
		}
	},
}

func init() {
	DdevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	DdevExecCmd.Flags().StringVarP(&execDirArg, "dir", "d", "", "Defines the execution directory within the container")
	DdevExecCmd.Flags().Bool("raw", true, "Use raw exec (do not interpret with bash inside container)")
	// This requires flags for exec to be specified prior to any arguments, allowing for
	// flags to be ignored by cobra for commands that are to be executed in a container.
	DdevExecCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(DdevExecCmd)
}
