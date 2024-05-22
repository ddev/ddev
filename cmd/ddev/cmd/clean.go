package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/amplitude"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var CleanCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "clean [projectname ...]",
	Short:             "Removes items DDEV has created",
	Long: `Stops all running projects and then removes downloads and snapshots
for the selected projects. Then clean will remove "ddev/ddev-*" images.

Warning - This command will permanently delete your snapshots for the named project[s].

Additional commands that can help clean up resources:
  ddev delete --omit-snapshot
  docker rmi -f $(docker images -q)
  docker system prune --volumes
	`,
	Example: `ddev clean
	ddev clean project1 project2
	ddev clean --all`,
	Run: func(cmd *cobra.Command, args []string) {

		// Make sure the user provides a project or flag
		cleanAll, _ := cmd.Flags().GetBool("all")
		if len(args) == 0 && !cleanAll {
			util.Failed("No project provided. See ddev clean --help for usage")
		}

		// Skip project validation
		originalRunValidateConfig := ddevapp.RunValidateConfig
		ddevapp.RunValidateConfig = false
		projects, err := getRequestedProjects(args, cleanAll)
		ddevapp.RunValidateConfig = originalRunValidateConfig

		if err != nil {
			util.Failed("Failed to get project(s) '%v': %v", args, err)
		}

		util.Warning("Warning - Snapshots for the following project[s] will be permanently deleted")

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

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			util.Warning("Dry run terminated without removing items")
			return
		}

		confirm := util.Prompt("Are you sure you want to continue? y/n", "n")
		if strings.ToLower(confirm) != "y" {
			return
		}

		util.Success("Powering off DDEV to avoid conflicts.")
		ddevapp.PowerOff()

		globalDdevDir := globalconfig.GetGlobalDdevDir()
		_ = os.RemoveAll(filepath.Join(globalDdevDir, "testcache"))
		_ = os.RemoveAll(filepath.Join(globalDdevDir, "bin"))

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

		output.UserOut.Print("Deleting Docker images that DDEV created...")
		err = deleteDdevImages(true)
		if err != nil {
			util.Failed("Failed to delete image tag", err)
		}
		util.Success("Finished cleaning DDEV projects")
	},
}

func init() {
	CleanCmd.Flags().BoolP("all", "a", false, "Clean all DDEV projects")
	CleanCmd.Flags().Bool("dry-run", false, "Run the clean command without deleting")
	RootCmd.AddCommand(CleanCmd)
}
