package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/globalconfig"
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
		powerOff()

		util.Success("Gathering ddev items to be cleaned")
		// Display a list of ddev projects that will be removed
		projectList := globalconfig.GetGlobalProjectList()
		fmt.Println("Snapshots and downloads from the following project will be removed")
		for appRoot := range projectList {
			fmt.Printf("\t Project %s will be cleaned \n", appRoot)
		}

		// Create a list of ddev images
		client := dockerutil.GetDockerClient()
		imagesToRemove, _ := client.ListImages(docker.ListImagesOptions{
			All: true,
		})

		// Provide space between sections for readability
		fmt.Println(" ")
		fmt.Println("The following ddev images will be removed")
		for _, image := range imagesToRemove {
			for _, tag := range image.RepoTags {
				if strings.HasPrefix(tag, "drud/ddev-") {
					fmt.Printf("\t %s \n", tag)
				}
			}
		}
		// Show the user a list of everything and ask to confirm
		if !util.Confirm("Confirm removal of the listed items?") {
			util.Warning("Program terminated")
			os.Exit(1)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			util.Warning("Dry run terminated without removing items")
			os.Exit(1)
		}

		ddevDir := globalconfig.GetGlobalDdevDir()
		_ = os.RemoveAll(filepath.Join(ddevDir, "cleantest"))
		// range projectList and remove them like this
		// _ = os.RemoveAll(filepath.Join(ddevDir, "testcache"))
		// _ = os.RemoveAll(filepath.Join(ddevDir, "bin"))

		// Delete the images in the list
		for _, image := range imagesToRemove {
			for _, tag := range image.RepoTags {
				if strings.HasPrefix(tag, "drud/ddev-") {
					_ = dockerutil.RemoveImage(tag)
				}
			}
		}

		util.Success("Finished cleaning ddev projects")
	},
}

func init() {
	CleanCmd.Flags().Bool("dry-run", false, "Do a dry run to see what will be removed")
	RootCmd.AddCommand(CleanCmd)
}
