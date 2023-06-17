package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// MutagenSyncCmd implements the ddev mutagen sync command
var MutagenSyncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "Explicit sync for mutagen",
	Example: `"ddev mutagen sync", "ddev mutagen sync <projectname>"`,
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""
		verbose := false
		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project-name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		if cmd.Flags().Changed("verbose") {
			verbose = true
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
			util.Failed("Failed to flush mutagen: %v", err)
		}
		if !verbose {
			return
		}

		_, _, longResult, _ := app.MutagenStatus()
		output.UserOut.Printf("%s", longResult)
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenSyncCmd)
	MutagenSyncCmd.Flags().Bool("verbose", false, "Extended/verbose output for mutagen status")
}
