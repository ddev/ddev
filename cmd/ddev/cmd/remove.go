package cmd

import (
	"fmt"

	"os"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var skipConfirmation bool

// LocalDevRMCmd represents the stop command
var LocalDevRMCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove an application's local services.",
	Long:    `Remove will delete the local service containers from this machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := getActiveApp()
		if err != nil {
			util.Failed("Failed to get active app: %v", err)
		}

		if app.SiteStatus() == "not found" {
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
