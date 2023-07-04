package cmd

import (
	"os"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	dockerImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

// DeleteImagesCmd implements the ddev delete images command
var DeleteImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Deletes ddev/ddev-* docker images not in use by current ddev version",
	Long:  "with --all it deletes all ddev/ddev-* docker images",
	Example: `ddev delete images
ddev delete images -y
ddev delete images --all`,

	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// If true, --yes, we don't stop and prompt before deletion
		deleteImagesNocConfirm, _ := cmd.Flags().GetBool("yes")
		if !deleteImagesNocConfirm {
			if !util.Confirm("Deleting unused ddev images. \nThis is a non-destructive operation, \nbut it may require that the images be downloaded again when you need them. \nOK to continue?") {
				os.Exit(1)
			}
		}
		util.Success("Powering off ddev to avoid conflicts")
		ddevapp.PowerOff()

		// The user can select to delete all ddev images.
		deleteAllImages, _ := cmd.Flags().GetBool("all")

		err := deleteDdevImages(deleteAllImages)
		if err != nil {
			util.Failed("Failed to delete image", err)
		}
		util.Success("All ddev images discovered were deleted.")
	},
}

func init() {
	DeleteImagesCmd.Flags().BoolP("yes", "y", false, "Yes - skip confirmation prompt")
	DeleteImagesCmd.Flags().BoolP("all", "a", false, "If set, deletes all Docker images created by ddev.")
	DeleteCmd.AddCommand(DeleteImagesCmd)
}

// deleteDdevImages removes Docker images prefixed with ddev-
func deleteDdevImages(deleteAll bool) error {

	client := dockerutil.GetDockerClient()
	images, err := client.ListImages(docker.ListImagesOptions{
		All: true,
	})
	if err != nil {
		return err
	}

	// If delete all images, find them by tag and return.
	if deleteAll {
		// Attempt to find ddev images by tag, searching for "drud" or "ddev" prefixes.
		// Some ddev images will not be found by this tag, future work will
		// be done to improve finding database images.
		for _, image := range images {
			for _, tag := range image.RepoTags {

				if strings.HasPrefix(tag, "drud/ddev-") || strings.HasPrefix(tag, "ddev/ddev-") {
					if err := dockerutil.RemoveImage(tag); err != nil {
						return err
					}
				}

			}
		}
		return nil
	}

	// Sort so that images that have -built on the end
	// come up before their parent images that don't
	sort.Slice(images, func(i, j int) bool {
		if images[i].RepoTags == nil || len(images[i].RepoTags) == 0 {
			return false
		}
		if images[j].RepoTags == nil || len(images[j].RepoTags) == 0 {
			return false
		}
		return images[i].RepoTags[0] > images[j].RepoTags[0]
	})

	webimg := dockerImages.GetWebImage()
	routerimage := dockerImages.GetRouterImage()
	sshimage := dockerImages.GetSSHAuthImage()

	nameAry := strings.Split(dockerImages.GetDBImage(nodeps.MariaDB, ""), ":")
	keepDBImageTag := "notagfound"
	if len(nameAry) > 1 {
		keepDBImageTag = nameAry[1]
	}

	// Too much code inside this loop, but complicated by multiple db images
	// and discrete names of images
	for _, image := range images {
		for _, tag := range image.RepoTags {
			// If anything has prefix "drud/" then delete it
			if strings.HasPrefix(tag, "drud/") {
				if err = dockerutil.RemoveImage(tag); err != nil {
					return err
				}
			}
			// If a webimage, but doesn't match our webimage, delete it
			if strings.HasPrefix(tag, versionconstants.WebImg) && !strings.HasPrefix(tag, webimg) && !strings.HasPrefix(tag, webimg+"-built") {
				if err = dockerutil.RemoveImage(tag); err != nil {
					return err
				}
			}
			if strings.HasPrefix(tag, "ddev/ddev-dbserver") && !strings.HasSuffix(tag, keepDBImageTag) && !strings.HasSuffix(tag, keepDBImageTag+"-built") {
				if err = dockerutil.RemoveImage(tag); err != nil {
					return err
				}
			}
			// TODO: Verify the functionality here. May not work since GetRouterImage() returns full image spec
			// If a routerImage, but doesn't match our routerimage, delete it
			if strings.HasPrefix(tag, dockerImages.GetRouterImage()) && !strings.HasPrefix(tag, routerimage) {
				if err = dockerutil.RemoveImage(tag); err != nil {
					return err
				}
			}
			// If a sshAgentImage, but doesn't match our sshAgentImage, delete it
			if strings.HasPrefix(tag, versionconstants.SSHAuthImage) && !strings.HasPrefix(tag, sshimage) && !strings.HasPrefix(tag, sshimage+"-built") {
				if err = dockerutil.RemoveImage(tag); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
