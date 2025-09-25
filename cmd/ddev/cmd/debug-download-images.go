package cmd

import (
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// If downloadAll is true, we'll download all images for all projects
var downloadAll bool

// DebugDownloadImagesCmd implements the ddev debug download-images command
var DebugDownloadImagesCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 0),
	Use:               "download-images [projectname ...]",
	Short:             "Download all images required by DDEV",
	Example: `ddev debug download-images
ddev debug download-images <project-name>
ddev debug download-images --all
`,
	Run: func(_ *cobra.Command, args []string) {
		if len(args) > 0 && downloadAll {
			util.Failed("Cannot specify project name with --all")
		}

		_, err := dockerutil.DownloadDockerComposeIfNeeded()
		if err != nil {
			util.Failed("Unable to download docker-compose: %v", err)
		}
		if globalconfig.DdevGlobalConfig.IsMutagenEnabled() {
			err = ddevapp.DownloadMutagenIfNeeded()
			if err != nil {
				util.Warning("Unable to download Mutagen: %v", err)
			}
		}

		var additionalImages []string

		// Skip project validation
		originalRunValidateConfig := ddevapp.RunValidateConfig
		ddevapp.RunValidateConfig = false
		projects, err := getRequestedProjects(args, downloadAll)
		ddevapp.RunValidateConfig = originalRunValidateConfig

		if err != nil {
			util.Success("Downloading basic images")
			additionalImages = []string{
				// Provide at least the default database image
				docker.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion),
			}
		} else {
			var projectNames []string
			for _, app := range projects {
				projectNames = append(projectNames, app.Name)
				_ = app.DockerEnv()
				err = app.WriteDockerComposeYAML()
				if err != nil {
					util.Failed("Failed to run `docker-compose config` for '%s': %v", app.Name, err)
				}
				appImages, err := app.FindAllImages()
				if err != nil {
					util.Failed("Failed to get images for '%s': %v", app.Name, err)
				}
				additionalImages = append(additionalImages, appImages...)
			}
			util.Success("Downloading images for %s", strings.Join(projectNames, ", "))
		}

		err = ddevapp.PullBaseContainerImages(additionalImages, true)
		if err != nil {
			util.Failed("Failed to pull DDEV images: %v", err)
		}

		util.Success("Successfully downloaded DDEV images")
	},
}

func init() {
	DebugDownloadImagesCmd.Flags().BoolVarP(&downloadAll, "all", "a", false, "Download all images for all projects")
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
