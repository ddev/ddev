package cmd

import (
	"fmt"

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
	Use:   "download-images [project]",
	Short: "Download all images required by DDEV",
	Example: `ddev debug download-images
ddev debug download-images <project-name>
ddev debug download-images --all
`,
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Args:              cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		projectName := ""
		if len(args) == 1 {
			if downloadAll {
				util.Failed("Cannot specify project name with --all")
			}
			projectName = args[0]
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

		additionalImages := map[string]string{}

		if downloadAll {
			util.Success("Downloading images for all projects")
			projects, err := ddevapp.GetProjects(false)
			if err != nil {
				util.Failed("Failed to get projects: %v", err)
			}
			for _, app := range projects {
				app.DockerEnv()
				err = app.WriteDockerComposeYAML()
				if err != nil {
					util.Warning("Failed to run `docker-compose config` for '%s': %v", app.Name, err)
					continue
				}
				appImages, err := app.FindAllImages()
				if err != nil {
					util.Warning("Failed to get images for '%s': %v", app.Name, err)
					continue
				}
				for k, v := range appImages {
					additionalImages[k] = v
				}
			}
		} else {
			app, err := ddevapp.GetActiveApp(projectName)
			if err == nil {
				util.Success("Downloading images for project '%s'", app.Name)
				app.DockerEnv()
				err = app.WriteDockerComposeYAML()
				if err != nil {
					util.Failed("Failed to run `docker-compose config` for '%s': %v", app.Name, err)
				}
				appImages, err := app.FindAllImages()
				if err != nil {
					util.Failed("Failed to get images for '%s': %v", app.Name, err)
				}
				additionalImages = appImages
			} else {
				util.Success("Downloading basic images")

				additionalImages = map[string]string{
					// Provide at least the default database image
					fmt.Sprintf("ddev-dbserver-%s-%s", nodeps.MariaDB, nodeps.MariaDBDefaultVersion): docker.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion),
				}
			}
		}

		err = ddevapp.PullBaseContainerImages(additionalImages, true)
		if err != nil {
			util.Failed("Failed to PullBaseContainerImages(): %v", err)
		}

		util.Success("Successfully downloaded DDEV images")
	},
}

func init() {
	DebugDownloadImagesCmd.Flags().BoolVarP(&downloadAll, "all", "a", false, "Download all images for all projects")
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
