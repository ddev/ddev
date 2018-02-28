package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevStopCmd represents the stop command
var DdevStopCmd = &cobra.Command{
	Use:   "stop [projectname]",
	Short: "Stop the development environment for a project.",
	Long: `Stop the development environment for a project. You can run 'ddev stop'
from a project directory to stop that project, or you can specify a running project
to stop by running 'ddev stop <projectname>.`,
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev stop' or 'ddev stop [projectname]'")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		app, err := ddevapp.GetActiveApp(siteName)
		if err != nil {
			util.Failed("Failed to stop: %v", err)
		}

		err = app.Stop()
		if err != nil {
			util.Failed("Failed to stop containers for %s: %v", app.GetName(), err)
		}

		util.Success("Project has been stopped.")
	},
}

func init() {
	RootCmd.AddCommand(DdevStopCmd)
}
