package cmd

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AddonReapplyCmd is the "ddev add-on reapply" command
var AddonReapplyCmd = &cobra.Command{
	Use:   "reapply <add-on> [add-on ...]",
	Short: "Re-run the pre/post-install actions of an installed add-on",
	Long:  `Re-run the pre-install and post-install actions of one or more installed add-ons without re-downloading or re-copying files. Useful when local configuration has changed and add-on logic needs to be re-applied.`,
	Example: `ddev add-on reapply my-addon
ddev add-on reapply my-addon another-addon
ddev add-on reapply --all
ddev add-on reapply --all --project my-project`,
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if !all && len(args) == 0 {
			return fmt.Errorf("requires at least 1 add-on name or --all flag")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")
		all, _ := cmd.Flags().GetBool("all")

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

		_ = app.DockerEnv()

		manifests, err := ddevapp.GatherAllManifests(app)
		if err != nil {
			util.Failed("Unable to gather manifests: %v", err)
		}

		var toReapply []ddevapp.AddonManifest
		if all {
			// Deduplicate: GatherAllManifests stores the same manifest under
			// multiple keys (name, repository, short repo name).
			seen := make(map[string]bool)
			for _, manifest := range manifests {
				if !seen[manifest.Name] {
					seen[manifest.Name] = true
					toReapply = append(toReapply, manifest)
				}
			}
		} else {
			for _, name := range args {
				manifest, ok := manifests[name]
				if !ok {
					util.Failed("The add-on '%s' does not seem to be installed. Use `ddev add-on list --installed` to see installed add-ons.", name)
				}
				toReapply = append(toReapply, manifest)
			}
		}

		for _, manifest := range toReapply {
			if len(manifest.PreInstallActions) == 0 && len(manifest.PostInstallActions) == 0 {
				util.Warning("Add-on '%s': no actions found in manifest. Re-install the add-on with `ddev add-on get` to enable reapply.", manifest.Name)
				continue
			}

			desc := ddevapp.InstallDesc{
				Name:          manifest.Name,
				Image:         manifest.Image,
				YamlReadFiles: manifest.YamlReadFiles,
			}

			util.Success("\nReapplying add-on %s:", manifest.Name)

			if len(manifest.PreInstallActions) > 0 {
				util.Success("Executing pre-install actions:")
			}
			for i, action := range manifest.PreInstallActions {
				err = ddevapp.ProcessAddonAction(action, desc, app, verbose)
				if err != nil {
					actionDesc := ddevapp.GetAddonDdevDescription(action)
					if !verbose {
						util.Failed("Could not process pre-install action (%d) '%s' for add-on '%s'.\nFor more detail, use --verbose", i, actionDesc, manifest.Name)
					} else {
						util.Failed("Could not process pre-install action (%d) '%s' for add-on '%s': %v", i, actionDesc, manifest.Name, err)
					}
				}
			}

			if len(manifest.PostInstallActions) > 0 {
				util.Success("Executing post-install actions:")
			}
			for i, action := range manifest.PostInstallActions {
				err = ddevapp.ProcessAddonAction(action, desc, app, verbose)
				if err != nil {
					actionDesc := ddevapp.GetAddonDdevDescription(action)
					if !verbose {
						util.Failed("Could not process post-install action (%d) '%s' for add-on '%s'.\nFor more detail, use --verbose", i, actionDesc, manifest.Name)
					} else {
						util.Failed("Could not process post-install action (%d) '%s' for add-on '%s': %v", i, actionDesc, manifest.Name, err)
					}
				}
			}
		}
	},
}

func init() {
	AddonReapplyCmd.Flags().BoolP("verbose", "v", false, "Extended/verbose output")
	AddonReapplyCmd.Flags().BoolP("all", "a", false, "Reapply all installed add-ons")
	AddonReapplyCmd.Flags().String("project", "", "Name of the project")
	_ = AddonReapplyCmd.RegisterFlagCompletionFunc("project", ddevapp.GetProjectNamesFunc("all", 0))
	AddonCmd.AddCommand(AddonReapplyCmd)
}
