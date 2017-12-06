package cmd

import (
	"fmt"
	"os"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// StartCmd represents the add command
var StartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"add"},
	Short:   "Start the local development environment for a site.",
	Long: `Start initializes and configures the web server and database containers to
provide a working environment for development.`,
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

func appStart() {
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		util.Failed("Failed to start: %v", err)
	}

	output.UserOut.Printf("Starting environment for %s...", app.GetName())

	err = app.Start()
	if err != nil {
		util.Failed("Failed to start %s: %v", app.GetName(), err)
	}

	var https bool
	web, err := app.FindContainerByType("web")
	if err == nil {
		https = dockerutil.CheckForHTTPS(web)
	}

	urlString := fmt.Sprintf("http://%s", app.HostName())
	if https {
		urlString = fmt.Sprintf("%s\nhttps://%s", urlString, app.HostName())
	}

	util.Success("Successfully started %s", app.GetName())
	util.Success("Your application can be reached at:\n%s", urlString)

}
func init() {
	RootCmd.AddCommand(StartCmd)
}
