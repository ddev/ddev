package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

var listCommandSettings = ddevapp.ListCommandSettings{}

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List projects",
	Long:    `List projects. Shows all projects by default, shows active projects only with --active-only`,
	Aliases: []string{"l", "ls"},
	Example: `ddev list
ddev list --active-only
ddev list -A
ddev list --type=drupal8
ddev list -t drupal8`,
	Run: func(cmd *cobra.Command, args []string) {
		ddevapp.List(listCommandSettings)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&listCommandSettings.ActiveOnly, "active-only", "A", false, "If set, only currently active projects will be displayed.")
	ListCmd.Flags().BoolVarP(&listCommandSettings.Continuous, "continuous", "", false, "If set, project information will be emitted until the command is stopped.")
	ListCmd.Flags().BoolVarP(&listCommandSettings.WrapTableText, "wrap-table", "W", false, "Display table with wrapped text if required.")
	ListCmd.Flags().StringVarP(&listCommandSettings.TypeFilter, "type", "t", "", "Show only projects of this type")
	ListCmd.Flags().IntVarP(&listCommandSettings.ContinuousSleepTime, "continuous-sleep-interval", "I", 1, "Time in seconds between ddev list --continuous output lists.")

	RootCmd.AddCommand(ListCmd)
}
