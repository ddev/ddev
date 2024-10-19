package cmd

import (
	"os"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AddonRemoveCmd is the "ddev add-on remove" command
var AddonRemoveCmd = &cobra.Command{
	Use:   "remove <addonOrURL>",
	Args:  cobra.ExactArgs(1),
	Short: "Remove a DDEV add-on which was previously installed in this project",
	Example: `ddev add-on remove someaddonname,
ddev add-on remove someowner/ddev-someaddonname,
ddev add-on remove ddev-someaddonname,
ddev add-on remove ddev-someaddonname --project my-project
`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp(cmd.Flag("project").Value.String())
		if err != nil {
			util.Failed("Unable to get project %v: %v", cmd.Flag("project").Value.String(), err)
		}

		origDir, _ := os.Getwd()

		defer func() {
			err = os.Chdir(origDir)
			if err != nil {
				util.Failed("Unable to chdir to %v: %v", origDir, err)
			}
		}()

		err = os.Chdir(app.GetConfigPath(""))
		if err != nil {
			util.Failed("Unable to chdir to %v: %v", app.GetConfigPath(""), err)
		}

		app.DockerEnv()

		err = ddevapp.RemoveAddon(app, args[0], nil, util.FindBashPath(), cmd.Flags().Changed("verbose"))
		if err != nil {
			util.Failed("Unable to remove add-on: %v", err)
		}
	},
}

func init() {
	AddonRemoveCmd.Flags().BoolP("verbose", "v", false, "Extended/verbose output")
	AddonRemoveCmd.Flags().String("project", "", "Name of the project to remove the add-on from")
	_ = AddonRemoveCmd.RegisterFlagCompletionFunc("project", ddevapp.GetProjectNamesFunc("all", 0))
	AddonCmd.AddCommand(AddonRemoveCmd)
}
