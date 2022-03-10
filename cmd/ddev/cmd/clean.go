package cmd

import (
	"os"
	"path/filepath"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

var CleanCmd = &cobra.Command{
	Use:     "clean",
	Short:   "Removes items ddev has created",
	Long:    "Removes downloads and snapshots from projects and removes images",
	Example: "ddev clean",
	Run: func(cmd *cobra.Command, args []string) {

		util.Success("Powering off ddev to avoid conflicts")
		// ddevapp.PowerOff()

		util.Warning("Warning - Snapshots for the following project(s) will be permanently deleted")

		cleanAll, _ := cmd.Flags().GetBool("all")
		projects, err := getRequestedProjects(args, cleanAll)
		if err != nil {
			util.Failed("Failed to get project(s): %v", err)
		}

		// Show the user how many snapshots per project that will be deleted
		for _, project := range projects {
			snapshots, err := project.ListSnapshots()
			if err != nil {
				util.Failed("Failed to get project %s snapshots: %v", project.Name, err)
			}
			if len(snapshots) > 0 {
				output.UserOut.Printf("%v Snapshots for project %s will be deleted", len(snapshots), project.Name)
			} else {
				output.UserOut.Printf("No snapshots found for project %s", project.Name)
			}
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			util.Warning("Dry run terminated without removing items")
			os.Exit(1)
		}

		if util.Confirm("Confirm removal of the listed items?") {
			ddevDir := globalconfig.GetGlobalDdevDir()
			_ = os.RemoveAll(filepath.Join(ddevDir, "testcache"))
			_ = os.RemoveAll(filepath.Join(ddevDir, "bin"))
			_ = os.RemoveAll(filepath.Join(ddevDir, "downloads"))

			output.UserOut.Print("Deleting snapshots and downloads for selected projects...")
			for _, project := range projects {
				// Delete snapshots and downloads for each project
				_ = os.RemoveAll(filepath.Join(project.AppRoot, ".ddev", "downloads"))
				_ = os.RemoveAll(filepath.Join(project.AppRoot, ".ddev", "db_snapshots"))
			}

			output.UserOut.Print("Deleting images that ddev created...")
			client := dockerutil.GetDockerClient()
			images, err := client.ListImages(docker.ListImagesOptions{
				All: true,
			})
			if err != nil {
				util.Failed("Failed to list images: %v", err)
			}
			err = deleteDdevImages(images)
			if err != nil {
				util.Failed("Failed to delete image tag", err)
			}
		}

		util.Success("Finished cleaning ddev projects")
	},
}

func init() {
	CleanCmd.Flags().BoolP("all", "a", false, "Clean all ddev projects")
	CleanCmd.Flags().Bool("dry-run", false, "Do a dry run to see what will be removed")
	RootCmd.AddCommand(CleanCmd)
}
