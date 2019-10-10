package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// DeleteImagesCmd implements the ddev delete images command
var DeleteImagesCmd = &cobra.Command{
	Use:     "images",
	Short:   "Delete docker images not currently in use",
	Example: `ddev delete images`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// This is were stuff goes
		if !util.Confirm("Deleting unused ddev images. \nThis is a non-destructive operation, \nbut it may require that the images be downloaded again when you need them. \nOK to continue?") {
			os.Exit(1)
		}
		util.Success("Powering off ddev to avoid conflicts")
		powerOff()

		client := dockerutil.GetDockerClient()

		images, err := client.ListImages(docker.ListImagesOptions{
			All: true,
		})
		if err != nil {
			util.Failed("Failed to list images: %v", err)
		}
		webimg := version.GetWebImage()
		dbaimage := version.GetDBAImage()
		routerimage := version.RouterImage + ":" + version.RouterTag
		sshimage := version.SSHAuthImage + ":" + version.SSHAuthTag

		dbImages := []string{}
		for v := range nodeps.ValidMariaDBVersions {
			dbImages = append(dbImages, version.GetDBImage(nodeps.MariaDB, v))
		}
		for v := range nodeps.ValidMySQLVersions {
			dbImages = append(dbImages, version.GetDBImage(nodeps.MySQL, v))
		}

		// Too much code inside this loop, but complicated by multiple db images
		// and discrete names of images
		for _, image := range images {
			for _, tag := range image.RepoTags {
				// If a webimage, but doesn't match our webimage, delete it
				if strings.HasPrefix(tag, version.WebImg) && !strings.HasPrefix(tag, webimg) {
					if err = removeImage(client, tag); err != nil {
						util.Failed("Failed to remove %s: %v", tag, err)
					}
				}
				// If a dbimage, but doesn't match our dbimages, delete it
				for _, imgName := range dbImages {
					if strings.HasPrefix(tag, imgName) && (tag != version.GetDBImage(nodeps.MariaDB) && tag != version.GetDBImage(nodeps.MySQL)) {
						if err = removeImage(client, tag); err != nil {
							util.Warning("Unable to remove %s: %v", tag, err)
						}
					}
				}
				// If a dbaimage, but doesn't match our dbaimage, delete it
				if strings.HasPrefix(tag, version.DBAImg) && !strings.HasPrefix(tag, dbaimage) {
					if err = removeImage(client, tag); err != nil {
						util.Failed("Failed to remove %s: %v", tag, err)
					}
				}
				// If a routerImage, but doesn't match our routerimage, delete it
				if strings.HasPrefix(tag, version.RouterImage) && !strings.HasPrefix(tag, routerimage) {
					if err = removeImage(client, tag); err != nil {
						util.Failed("Failed to remove %s: %v", tag, err)
					}
				}
				// If a sshAgentImage, but doesn't match our sshAgentImage, delete it
				if strings.HasPrefix(tag, version.SSHAuthImage) && !strings.HasPrefix(tag, sshimage) {
					if err = removeImage(client, tag); err != nil {
						util.Failed("Failed to remove %s: %v", tag, err)
					}
				}
			}
		}
	},
}

func init() {
	DeleteCmd.AddCommand(DeleteImagesCmd)
}

func removeImage(client *docker.Client, tag string) error {
	util.Warning("Removing container: %s", tag)
	err := client.RemoveImage(tag)
	if err != nil {
		util.Failed("Failed to remove %s: %v", tag, err)
	}
	return nil
}
