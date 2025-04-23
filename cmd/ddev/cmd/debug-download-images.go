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

		imagesPulled, err := (&ddevapp.DdevApp{}).PullContainerImages()
		if err != nil {
			util.Failed("Failed to PullContainerImages(): %v", err)
		}

		if imagesPulled {
			util.Success("Successfully downloaded DDEV images")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
