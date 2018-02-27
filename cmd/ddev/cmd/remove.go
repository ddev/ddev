package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var removeData bool

// DdevRemoveCmd represents the remove command
var DdevRemoveCmd = &cobra.Command{
	Use:     "remove [sitename]",
	Aliases: []string{"rm"},
	Short:   "Remove the development environment for a site.",
	Long: `Remove the development environment for a site. You can run 'ddev remove'
from a site directory to remove that site, or you can specify a site to remove
by running 'ddev remove <sitename>'. By default, remove is a non-destructive operation and will
leave database contents intact. Remove never touches your code or files directories.

To remove database contents, you may use the --remove-data flag with remove.`,
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev remove' or 'ddev remove [appname]'")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		app, err := ddevapp.GetActiveApp(siteName)
		if err != nil {
			util.Failed("Failed to remove: %v", err)
		}

		if app.SiteStatus() == ddevapp.SiteNotFound {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		err = app.Down(removeData)
		if err != nil {
			util.Failed("Failed to remove %s: %s", app.GetName(), err)
		}

		util.Success("Successfully removed the %s project.", app.GetName())
	},
}

func init() {
	DdevRemoveCmd.Flags().BoolVarP(&removeData, "remove-data", "R", false, "Remove stored project data (MySQL, logs, etc.)")
	RootCmd.AddCommand(DdevRemoveCmd)
}
