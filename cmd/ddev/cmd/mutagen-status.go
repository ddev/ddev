package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// MutagenStatusCmd implements the ddev mutagen status command
var MutagenStatusCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "status",
	Short:             "Shows Mutagen sync status",
	Aliases:           []string{"st"},
	Example:           `"ddev mutagen status", "ddev mutagen status <projectname>"`,
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
		status, shortResult, _, err := app.MutagenStatus()

		if err != nil {
			util.Failed("Unable to get Mutagen status for project %s, output='%s': %v", app.Name, shortResult, err)
		}
		ok := "Mutagen: " + status
		resultOut := shortResult
		if verbose {
			fullResult, err := exec.RunHostCommand(globalconfig.GetMutagenPath(), "sync", "list", "-l", ddevapp.MutagenSyncName(app.Name))
			if err != nil {
				util.Failed("Unable to get Mutagen status for project %s, output='%s': %v", app.Name, fullResult, err)
			}

			resultOut = "\n" + fullResult
		}
		output.UserOut.Printf("%s: %s", ok, resultOut)
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenStatusCmd)
	MutagenStatusCmd.Flags().BoolP("verbose", "l", false, "Extended verbose output for Mutagen status")
}
