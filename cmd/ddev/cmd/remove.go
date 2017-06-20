package cmd

import (
	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var removeData bool

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:     "remove [sitename]",
	Aliases: []string{"rm"},
	Short:   "Remove the local development environment for a site.",
	Long: `Remove the local development environment for a site. You can run 'ddev remove'
from a site directory to remove that site, or you can specify a site to remove
by running 'ddev rm <sitename>. By default, remove is a non-destructive operation and will
leave database contents intact.

To remove database contents, you may use the --remove-data flag with remove.`,
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

		err = app.Down(removeData)
		if err != nil {
			util.Failed("Failed to remove %s: %s", app.GetName(), err)
		}

		util.Success("Successfully removed the %s application.\n", app.GetName())
	},
}

func init() {
	LocalDevRMCmd.Flags().BoolVarP(&removeData, "remove-data", "R", false, "Remove stored application data (MySQL, logs, etc.)")
	RootCmd.AddCommand(LocalDevRMCmd)
}
