package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// continuous, if set, makes list continuously output
var continuous bool

// activeOnly, if set, shows only running projects
var activeOnly bool

// continuousSleepTime is time to sleep between reads with --continuous
var continuousSleepTime = 1

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  `List projects. Shows all projects by default, shows active projects only with --active-only`,
	Example: `ddev list
ddev list --active-only
ddev list -A`,
	Run: func(cmd *cobra.Command, args []string) {
		ddevapp.List(activeOnly, continuous, continuousSleepTime)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&activeOnly, "active-only", "A", false, "If set, only currently active projects will be displayed.")
	ListCmd.Flags().BoolVarP(&continuous, "continuous", "", false, "If set, project information will be emitted until the command is stopped.")
	ListCmd.Flags().IntVarP(&continuousSleepTime, "continuous-sleep-interval", "I", 1, "Time in seconds between ddev list --continous output lists.")

	RootCmd.AddCommand(ListCmd)
}
