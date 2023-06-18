package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"strings"
)

// MutagenResetCmd implements the ddev mutagen reset command
var MutagenResetCmd = &cobra.Command{
	Use:     "reset",
	Short:   "Reset mutagen for project",
	Long:    "Stops project, removes the mutagen docker volume",
	Example: `"ddev mutagen reset", "ddev mutagen reset <projectname>"`,
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

		err = app.MutagenSyncFlush()
		if err != nil {
			// if the mutagen session just did not exist, continue on
			if !strings.Contains(err.Error(), "does not exist") {
				util.Warning("Could not flush mutagen: %v", err)
			}
		}

		err = ddevapp.MutagenReset(app)
		if err != nil {
			util.Failed("Could not reset mutagen: %v", err)
		}
		util.Success("Mutagen has been reset. You may now `ddev start` with or without mutagen enabled.")
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenResetCmd)
}
