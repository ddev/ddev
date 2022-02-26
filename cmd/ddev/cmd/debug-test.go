package cmd

import (
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// DebugTestCmdCmd implements the ddev debug test command
var DebugTestCmdCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run diagnostics on ddev using the test_ddev.sh script",
	Example: "ddev debug test",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}
		tmpDir := os.TempDir()
		bashPath := util.FindBashPath()
		err := fileutil.CopyEmbedAssets(bundledAssets, "scripts", tmpDir)
		if err != nil {
			util.Failed("Failed to copy test_ddev.sh to %s: %v", tmpDir, err)
		}

		path := os.Getenv("PATH")
		_ = path
		dDebug := os.Getenv("DDEV_DEBUG")
		_ = dDebug
		c := []string{"-c", filepath.Join(tmpDir, "test_ddev.sh")}
		util.Success("Running %s %v", bashPath, c)
		err = exec.RunInteractiveCommand(bashPath, c)
		if err != nil {
			util.Failed("Failed running test_ddev.sh: %v\n. You can run it manually with `curl -sL -O https://raw.githubusercontent.com/drud/ddev/master/scripts/test_ddev.sh && bash test_ddev.sh`", err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCmdCmd)
}
