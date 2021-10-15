package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
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

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to debug download-images: %v", err)
		}
		err = app.Init("")
		if err != nil {
			util.Failed("Failed to init: %v", err)
		}
		err = app.PullContainerImages()
		if err != nil {
			util.Failed("Failed to debug download-images: %v", err)
		}

		util.Success("Successfully downloaded ddev images")
	},
}

func init() {
	DebugCmd.AddCommand(DebugDownloadImagesCmd)
}
