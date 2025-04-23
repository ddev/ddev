package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugDownloadImagesCmd implements the ddev debug download-images command
var DebugDownloadImagesCmd = &cobra.Command{
	Use:     "download-images",
	Short:   "Download all images required by DDEV",
	Example: "ddev debug download-images",
	Run: func(_ *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}
		_, err := dockerutil.DownloadDockerComposeIfNeeded()
		if err != nil {
			util.Warning("Unable to download docker-compose: %v", err)
		}
		if globalconfig.DdevGlobalConfig.IsMutagenEnabled() {
			err = ddevapp.DownloadMutagenIfNeeded()
			if err != nil {
				util.Warning("Unable to download Mutagen: %v", err)
			}
		}

		app, err := ddevapp.GetActiveApp("")
		var imagesPulled bool

		// If we're in a project directory, use the app context
		if err == nil {
			util.Success("Downloading images for project '%s'", app.Name)

			app.DockerEnv()
			if err = app.WriteDockerComposeYAML(); err != nil {
				util.Failed("Failed to get compose-config: %v", err)
			}

			imagesPulled, err = app.PullContainerImages()
			if err != nil {
				util.Failed("Failed to pull container images: %v", err)
			}
		} else {
			util.Warning("Downloading common images")
			imagesPulled, err = (&ddevapp.DdevApp{}).PullContainerImages()
			if err != nil {
				util.Failed("Failed to pull images: %v", err)
			}
		}

		if imagesPulled {
			util.Success("Successfully downloaded DDEV images")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
