package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugTestCmdCmd implements the ddev debug test command
var DebugTestCmdCmd = &cobra.Command{
	Use:     "test",
	Short:   "Run diagnostics on DDEV using the embedded test_ddev.sh script",
	Example: "ddev debug test",
	Run: func(_ *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}
		tmpDir := os.TempDir()
		outputFilename := filepath.Join(tmpDir, "ddev-debug-test.txt")
		bashPath := util.FindBashPath()
		err := fileutil.CopyEmbedAssets(bundledAssets, "scripts", tmpDir)
		if err != nil {
			util.Failed("Failed to copy test_ddev.sh to %s: %v", tmpDir, err)
		}
		p := dockerutil.MassageWindowsHostMountpoint(tmpDir)
		c := []string{"-c", path.Join(p, "test_ddev.sh") + " " + outputFilename}
		util.Success("Running %s %v", bashPath, c)

		// Create a new file to capture output
		f, err := os.Create(outputFilename)
		if err != nil {
			util.Failed("Failed to create output file: %v", err)
		}
		defer f.Close()

		activeApps := ddevapp.GetActiveProjects()
		if len(activeApps) > 0 {
			y := util.Confirm("OK to stop running projects? This does no harm, and they will be restarted")
			if !y {
				util.Warning("Exiting, no permission given to poweroff")
				os.Exit(3)
			}
		}
		util.Success("Doing ddev poweroff but will restart projects at completion")
		ddevapp.PowerOff()

		// Use MultiWriter to write to both file and stdout
		mw := io.MultiWriter(os.Stdout, f)

		testErr := exec.RunInteractiveCommandWithOutput(bashPath, c, mw)
		util.Success("Restarting previously-running DDEV projects")
		for _, app := range activeApps {
			_ = app.Start()
		}
		util.Success("Output file written to:\n%s\nPlease provide the file for support in Discord or the issue queue.", outputFilename)
		if testErr != nil {
			util.Failed("Failed running test_ddev.sh: %v\n. You can run it manually with `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/master/cmd/ddev/cmd/scripts/test_ddev.sh && bash test_ddev.sh`", testErr)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCmdCmd)
}
