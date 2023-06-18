package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// MutagenMonitorCmd implements the ddev mutagen monitor command
var MutagenMonitorCmd = &cobra.Command{
	Use:     "monitor",
	Short:   "Monitor mutagen status",
	Example: `"ddev mutagen monitor", "ddev mutagen monitor <projectname>"`,
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""
		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project-name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}
		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get active project: %v", err)
		}
		if !(app.IsMutagenEnabled()) {
			util.Warning("Mutagen is not enabled on project %s", app.Name)
			return
		}
		ddevapp.MutagenMonitor(app)
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenMonitorCmd)
}
