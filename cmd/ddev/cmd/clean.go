package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var CleanCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "clean [projectname ...]",
	Short:             "Removes items DDEV has created",
	Long: `Stops all running projects and removes downloads and snapshots for the selected projects.
If --all flag is used, removes all "ddev/ddev-" images, otherwise removes only unused images.

Warning: this command will permanently delete your snapshots for the named project(s).

Additional commands that can help clean up resources:
  ddev delete --omit-snapshot
  docker rmi -f $(docker images -q)
  docker system prune --volumes
  docker builder prune
	`,
	Example: heredoc.DocI2S(`
		ddev clean
		ddev clean project1 project2
		ddev clean --all
	`),
	Run: func(cmd *cobra.Command, args []string) {
		cleanAll, _ := cmd.Flags().GetBool("all")

		// Skip project validation
		originalRunValidateConfig := ddevapp.RunValidateConfig
		ddevapp.RunValidateConfig = false
		projects, err := getRequestedProjects(args, cleanAll)
		ddevapp.RunValidateConfig = originalRunValidateConfig

		if err != nil {
			util.Failed("Failed to get project(s) '%v': %v", strings.Join(args, ", "), err)
		}

		util.Warning("Warning: snapshots for the following project(s) will be permanently deleted:\n")

		// Show the user how many snapshots per project that will be deleted
		for _, project := range projects {
			snapshots, err := project.ListSnapshots()
			if err != nil {
				util.Failed("Failed to get project %s snapshots: %v", project.Name, err)
			}
			if len(snapshots) > 0 {
				output.UserOut.Printf("%v snapshots for project %s will be deleted", len(snapshots), project.Name)
			} else {
				output.UserOut.Printf("No snapshots found for project %s", project.Name)
			}
		}

		output.UserOut.Println()

		globalDdevDir := globalconfig.GetGlobalDdevDir()
		globalDirectoriesToRemove := []string{
			filepath.Join(globalDdevDir, "bin"),
			filepath.Join(globalDdevDir, "testcache"),
		}

		if cleanAll {
			dirList, _ := util.ArrayToReadableOutput(globalDirectoriesToRemove)
			util.Warning("Warning: the following directories will be removed:")
			output.UserOut.Printf("%s", dirList)
		}

		err = deleteDdevImages(cleanAll, true)
		if err != nil {
			util.Failed("Failed to delete images: %v", err)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if dryRun {
			util.Warning("Dry run terminated without removing items")
			return
		}

		cleanImagesNoConfirm, _ := cmd.Flags().GetBool("yes")

		if !cleanImagesNoConfirm {
			if !util.ConfirmTo("Are you sure you want to continue?", false) {
				if globalconfig.IsInteractive() {
					util.Failed("User cancelled operation. Terminating without removing items.")
				} else {
					util.Failed("Unable to continue because DDEV_NONINTERACTIVE=true, use `--yes` flag to proceed.")
				}
			}
		}

		if needsPoweroffToDeleteImages {
			util.Success("Powering off DDEV to avoid conflicts.")
			ddevapp.PowerOff()
		} else {
			for _, project := range projects {
				err = project.Stop(false, false)
				if err != nil {
					util.Warning("Failed to stop project %s: %v", project.Name, err)
				}
			}
		}

		if cleanAll {
			for _, dir := range globalDirectoriesToRemove {
				_ = os.RemoveAll(dir)
			}
		}

		amplitude.Clean()

		output.UserOut.Print("Deleting snapshots and downloads for selected projects...")
		for _, project := range projects {
			// Delete snapshots and downloads for each project
			err = os.RemoveAll(project.GetConfigPath(".downloads"))
			if err != nil {
				util.Warning("There was an error removing .downloads for project %v", project)
			}
			err = os.RemoveAll(project.GetConfigPath("db_snapshots"))
			if err != nil {
				util.Warning("There was an error removing db_snapshots for project %v", project)
			}
		}

		if cleanAll {
			output.UserOut.Print("Deleting all Docker images that DDEV created...")
		} else {
			output.UserOut.Print("Deleting unused Docker images that DDEV created...")
		}
		err = deleteDdevImages(cleanAll, false)
		if err != nil {
			util.Warning("Failed to delete images: %v", err)
		}

		util.Success("Finished cleaning DDEV projects.")
		util.Success("Optionally, run `docker builder prune` to clean unused builder cache.")
	},
}

func init() {
	CleanCmd.Flags().BoolP("all", "a", false, "Clean all DDEV projects")
	CleanCmd.Flags().Bool("dry-run", false, "Run the clean command without deleting")
	CleanCmd.Flags().BoolP("yes", "y", false, "Yes - skip confirmation prompt")
	RootCmd.AddCommand(CleanCmd)
}
