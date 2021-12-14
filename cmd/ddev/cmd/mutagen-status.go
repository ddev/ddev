package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// MutagenStatusCmd implements the ddev mutagen status command
var MutagenStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Shows mutagen sync status",
	Example: `"ddev mutagen status", "ddev mutagen status <projectname>"`,
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
		status, shortResult, longResult, _ := app.MutagenStatus()

		ok := "Mutagen: " + status
		resultOut := shortResult
		if verbose {
			resultOut = "\n" + longResult
		}
		output.UserOut.Printf("%s: %s", ok, resultOut)
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenStatusCmd)
	MutagenStatusCmd.Flags().Bool("verbose", false, "Extended/verbose output for mutagen status")
}
