package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"runtime"
)

// DebugDownloadImagesCmd implements the ddev debug download-images command
var DebugDownloadImagesCmd = &cobra.Command{
	Use:     "download-images",
	Short:   "Download all images required by ddev",
	Example: "ddev debug download-images",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}
		_, err := dockerutil.DownloadDockerComposeIfNeeded()
		if err != nil {
			util.Warning("Unable to download docker-compose: %v", err)
		}
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			err = ddevapp.DownloadMutagenIfNeeded()
			if err != nil {
				util.Warning("Unable to download mutagen: %v", err)
			}
		}

		err = ddevapp.PullBaseContainerImages()
		if err != nil {
			util.Failed("Failed to PullBaseContainerImages(): %v", err)
		}

		// Provide at least the default database image
		dbImage := versionconstants.GetDBImage(nodeps.MariaDB, nodeps.MariaDBDefaultVersion)
		err = dockerutil.Pull(dbImage)
		if err != nil {
			util.Failed("Failed to pull dbImage: %v", err)
		}
		util.Debug("Pulled %s", dbImage)

		util.Success("Successfully downloaded ddev images")
	},
}

func init() {
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
