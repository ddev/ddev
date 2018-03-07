package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// StartCmd represents the add command
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"add"},
	Short:   "Start a ddev project.",
	Long: `Start initializes and configures the web server and database containers to
provide a local development environment.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(0)
		}

		dockerutil.EnsureDdevNetwork()
	},
	Run: func(cmd *cobra.Command, args []string) {
		appStart()
	},
}

// appStart is a convenience function to encapsulate startup functionality
func appStart() {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Failed("Failed to start project: %v", err)
	}

	output.UserOut.Printf("Starting environment for %s...", app.GetName())

	err = app.Start()
	if err != nil {
		util.Failed("Failed to start %s: %v", app.GetName(), err)
	}

	util.Success("Successfully started %s", app.GetName())
	util.Success("Your project can be reached at %s and %s", app.GetHTTPURL(), app.GetHTTPSURL())

}
func init() {
	RootCmd.AddCommand(StartCmd)
}
