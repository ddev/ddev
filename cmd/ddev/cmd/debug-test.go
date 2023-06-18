package cmd

import (
	"os"
	"path"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugTestCmdCmd implements the ddev debug test command
var DebugTestCmdCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run diagnostics on ddev using the embedded test_ddev.sh script",
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
		p := dockerutil.MassageWindowsHostMountpoint(tmpDir)
		c := []string{"-c", path.Join(p, "test_ddev.sh")}
		util.Success("Running %s %v", bashPath, c)
		err = exec.RunInteractiveCommand(bashPath, c)
		if err != nil {
			util.Failed("Failed running test_ddev.sh: %v\n. You can run it manually with `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/master/cmd/ddev/cmd/scripts/test_ddev.sh && bash test_ddev.sh`", err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCmdCmd)
}
