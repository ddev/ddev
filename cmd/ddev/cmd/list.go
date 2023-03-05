package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// continuous, if set, makes list continuously output
var continuous bool

// activeOnly, if set, shows only running projects
var activeOnly bool

// continuousSleepTime is time to sleep between reads with --continuous
var continuousSleepTime = 1

// wrapListTable allow that the text in the table of ddev list wraps instead of cutting it to fit the terminal width
var wrapListTableText bool

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
		ddevapp.List(activeOnly, continuous, wrapListTableText, continuousSleepTime)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&activeOnly, "active-only", "A", false, "If set, only currently active projects will be displayed.")
	ListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted until the command is stopped.")
	ListCmd.Flags().BoolVarP(&wrapListTableText, "wrap-table", "W", false, "Display table with wrapped text if required.")
	ListCmd.Flags().IntVarP(&continuousSleepTime, "continuous-sleep-interval", "I", 1, "Time in seconds between ddev list --continuous output lists.")

	RootCmd.AddCommand(ListCmd)
}
