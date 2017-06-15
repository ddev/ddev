package cmd

import (
	"fmt"

	"os"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var skipConfirmation bool

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:     "remove [sitename]",
	Aliases: []string{"rm"},
	Short:   "Remove the local development environment for a site. (Destructive)",
	Long: `Remove the local development environment for a site. You can run 'ddev remove'
from a site directory to remove that site, or you can specify a site to remove
by running 'ddev stop <sitename>. Remove is a destructive operation. It will
remove all containers for the site, destroying database contents in the process.
Your project code base and files will not be affected.`,
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use `ddev remove` or `ddev remove [appname]`")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		app, err := getActiveApp(siteName)
		if err != nil {
			util.Failed("Failed to get active app: %v", err)
		}

		if app.SiteStatus() == platform.SiteNotFound {
			util.Failed("App not running locally. Try `ddev start`.")
		}

		if !skipConfirmation {
			fmt.Printf("Is it ok to remove the site %s with all of its containers? All data will be lost. (y/N): ", app.GetName())
			if !util.AskForConfirmation() {
				util.Warning("App removal canceled by user.")
				os.Exit(2)
			}
		}
		err = app.Down()
		if err != nil {
			util.Failed("Failed to remove %s: %s", app.GetName(), err)
		}

		util.Success("Successfully removed the %s application.\n", app.GetName())
	},
}

func init() {
	LocalDevRMCmd.Flags().BoolVarP(&skipConfirmation, "skip-confirmation", "y", false, "Skip confirmation step.")
	RootCmd.AddCommand(LocalDevRMCmd)
}
