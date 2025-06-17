package cmd

import (
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	dockerImage "github.com/docker/docker/api/types/image"
	"github.com/spf13/cobra"
)

var hasImagesToRemove = true

// DeleteImagesCmd implements the ddev delete images command
var DeleteImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Deletes ddev/ddev-* Docker images not in use by current DDEV version",
	Long:  "with --all it deletes all ddev/ddev-* Docker images",
	Example: heredoc.DocI2S(`
		ddev delete images
		ddev delete images -y
		ddev delete images --all
	`),
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		// If true, --yes, we don't stop and prompt before deletion
		deleteImagesNoConfirm, _ := cmd.Flags().GetBool("yes")
		// The user can select to delete all ddev images
		deleteAllImages, _ := cmd.Flags().GetBool("all")

		err := deleteDdevImages(deleteAllImages, true)
		if err != nil {
			util.Failed("Failed to delete images: %v", err)
		}

		if !hasImagesToRemove {
			if deleteAllImages {
				util.Success("No DDEV images found.")
			} else {
				util.Success("No unused DDEV images found.")
			}
			return
		}

		if !deleteImagesNoConfirm {
			if !util.Confirm("OK to continue?") {
				os.Exit(1)
			}
		}

		util.Success("Powering off DDEV to avoid conflicts")
		ddevapp.PowerOff()

		if err := deleteDdevImages(deleteAllImages, false); err != nil {
			util.Failed("Failed to delete images: %v", err)
		}
		util.Success("All DDEV images discovered were deleted.")
	},
}

func init() {
	DeleteImagesCmd.Flags().BoolP("yes", "y", false, "Yes - skip confirmation prompt")
	DeleteImagesCmd.Flags().BoolP("all", "a", false, "If set, deletes all Docker images created by DDEV.")
	DeleteCmd.AddCommand(DeleteImagesCmd)
}

// deleteDdevImages removes Docker images prefixed with "ddev/ddev-", or "drud/ddev-",
// or com.docker.compose.project starting with "ddev-"
// If dryRun is true, it only prints images to be deleted without removing them.
func deleteDdevImages(deleteAll, dryRun bool) error {
	ctx, client := dockerutil.GetDockerClient()

	allImages, err := client.ImageList(ctx, dockerImage.ListOptions{
		All: true,
	})
	if err != nil {
		return err
	}

	var images []dockerImage.Summary
	for _, image := range allImages {
		exists := false
		if len(image.RepoTags) > 0 {
			for _, tag := range image.RepoTags {
				if strings.HasPrefix(tag, "ddev/ddev-") || strings.HasPrefix(tag, "drud/ddev-") {
					images = append(images, image)
					exists = true
					break
				}
			}
		}
		if !exists {
			if projectName, ok := image.Labels["com.docker.compose.project"]; ok && strings.HasPrefix(projectName, "ddev-") {
				images = append(images, image)
			}
		}
	}

	// Sort for more readable output
	sort.Slice(images, func(i, j int) bool {
		labelI, okI := images[i].Labels["com.docker.compose.project"]
		labelJ, okJ := images[j].Labels["com.docker.compose.project"]

		if okI && okJ && labelI != labelJ {
			return labelI < labelJ
		}
		if okI && !okJ {
			return true
		}
		if !okI && okJ {
			return false
		}

		var tagI, tagJ string
		if len(images[i].RepoTags) > 0 {
			tagI = images[i].RepoTags[0]
		}
		if len(images[j].RepoTags) > 0 {
			tagJ = images[j].RepoTags[0]
		}
		return tagI < tagJ
	})

	if len(images) == 0 {
		hasImagesToRemove = false
		return nil
	}

	if deleteAll {
		if dryRun {
			util.Warning("Warning: the following %d Docker image(s) will be deleted:", len(images))
		}
		for _, image := range images {
			projectName := image.Labels["com.docker.compose.project"]
			imageName := image.ID
			if len(image.RepoTags) > 0 {
				imageName = strings.Join(image.RepoTags, ", ")
			}
			if dryRun {
				output.UserOut.Printf("Image to be deleted%s: %s",
					func() string {
						if projectName != "" {
							return " from " + projectName
						}
						return ""
					}(),
					imageName,
				)
				continue
			}
			if err := dockerutil.RemoveImage(image.ID); err != nil {
				return err
			}
		}
		if dryRun {
			util.Warning("Deleting images is a non-destructive operation.")
			util.Warning("You may need to download images again when you need them.\n")
		}
		return nil
	}

	allProjects, err := ddevapp.GetProjects(false)
	if err != nil {
		return err
	}

	projectMap := make(map[string]*ddevapp.DdevApp, len(allProjects))
	var composeProjectNames []string
	for _, project := range allProjects {
		name := project.GetComposeProjectName()
		projectMap[name] = project
		composeProjectNames = append(composeProjectNames, name)
	}

	webImage := ddevImages.GetWebImage()
	dbImagePrefix := versionconstants.DBImg
	dbImageSuffix := versionconstants.BaseDBTag
	routerImage := ddevImages.GetRouterImage()
	sshImage := ddevImages.GetSSHAuthImage()
	xhguiImage := ddevImages.GetXhguiImage()
	utilitiesImage := versionconstants.UtilitiesImage

	var filteredImages []dockerImage.Summary
	for _, image := range images {
		projectName := image.Labels["com.docker.compose.project"]
		// Remove images from unlisted or not properly deleted projects
		if projectName != "" && projectName != ddevapp.SSHAuthName && !slices.Contains(composeProjectNames, projectName) {
			filteredImages = append(filteredImages, image)
			continue
		}

		skip := false
		var serviceNames []string
		if slices.Contains(composeProjectNames, projectName) {
			if app := projectMap[projectName]; app != nil {
				if services, ok := app.ComposeYaml["services"].(map[string]interface{}); ok {
					for serviceName := range services {
						serviceNames = append(serviceNames, serviceName)
					}
				}
			}
		}

		for _, tag := range image.RepoTags {
			if projectName != "" {
				for _, serviceName := range serviceNames {
					if strings.Contains(tag, serviceName) {
						skip = true
						break
					}
				}
			}
			if skip {
				break
			}

			if tag == webImage ||
				(strings.HasPrefix(tag, webImage) && strings.HasSuffix(tag, "-built")) ||
				(strings.HasPrefix(tag, dbImagePrefix) && strings.HasSuffix(tag, dbImageSuffix)) ||
				(strings.HasPrefix(tag, dbImagePrefix) && strings.Contains(tag, dbImageSuffix) && strings.HasSuffix(tag, "-built")) ||
				tag == routerImage ||
				tag == sshImage ||
				(strings.HasPrefix(tag, sshImage) && strings.HasSuffix(tag, "-built")) ||
				tag == xhguiImage ||
				strings.HasPrefix(tag, utilitiesImage) {
				skip = true
				break
			}
		}
		if !skip {
			filteredImages = append(filteredImages, image)
		}
	}

	if len(filteredImages) == 0 {
		hasImagesToRemove = false
		return nil
	}

	if dryRun {
		util.Warning("Warning: the following %d Docker image(s) will be deleted:", len(filteredImages))
	}

	for _, image := range filteredImages {
		projectName := image.Labels["com.docker.compose.project"]
		imageName := image.ID
		if len(image.RepoTags) > 0 {
			imageName = strings.Join(image.RepoTags, ", ")
		}
		if dryRun {
			output.UserOut.Printf("Image to be deleted%s: %s",
				func() string {
					if projectName != "" {
						return " from " + projectName
					}
					return ""
				}(),
				imageName,
			)
			continue
		}
		if err := dockerutil.RemoveImage(image.ID); err != nil {
			return err
		}
	}

	if dryRun {
		util.Warning("Deleting images is a non-destructive operation.")
		util.Warning("You may need to download images again when you need them.\n")
	}

	return nil
}
