package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

var ListCommandSettings = ddevapp.ListCommandSettings{}

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List projects",
	Long:    `List projects. Shows all projects by default, shows active projects only with --active-only`,
	Aliases: []string{"l", "ls"},
	Example: `ddev list
ddev list --active-only
ddev list -A`,
	Run: func(cmd *cobra.Command, args []string) {
		ddevapp.List(ListCommandSettings)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&ListCommandSettings.ActiveOnly, "active-only", "A", false, "If set, only currently active projects will be displayed.")
	ListCmd.Flags().BoolVarP(&ListCommandSettings.Continuous, "continuous", "", false, "If set, project information will be emitted until the command is stopped.")
	ListCmd.Flags().BoolVarP(&ListCommandSettings.WrapTableText, "wrap-table", "W", false, "Display table with wrapped text if required.")
	ListCmd.Flags().StringVarP(&ListCommandSettings.TypeFilter, "type", "t", "", "Show only projects of this type")
	ListCmd.Flags().IntVarP(&ListCommandSettings.ContinuousSleepTime, "continuous-sleep-interval", "I", 1, "Time in seconds between ddev list --continuous output lists.")

	RootCmd.AddCommand(ListCmd)
}
