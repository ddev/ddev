package cmd

import (
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/output"
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
		outputFilename = filepath.ToSlash(outputFilename)
		bashPath := util.FindBashPath()
		err := fileutil.CopyEmbedAssets(bundledAssets, "scripts", tmpDir, nil)
		if err != nil {
			util.Failed("Failed to copy test_ddev.sh to %s: %v", tmpDir, err)
		}
		p := util.WindowsPathToCygwinPath(tmpDir)
		c := []string{"-c", path.Join(p, "test_ddev.sh") + " " + outputFilename}
		util.Debug("Running %s %v", bashPath, c)

		// Create a new file to capture output
		f, err := os.Create(outputFilename)
		if err != nil {
			util.Failed("Failed to create output file: %v", err)
		}
		defer f.Close()

		util.Success("Please make sure you have already looked at troubleshooting guide:")
		util.Success("https://ddev.readthedocs.io/en/stable/users/usage/troubleshooting/")
		util.Success("Simple things to check:\n* Use latest stable DDEV version\n* ddev poweroff\n* Restart Docker Provider\n* Reboot computer\n* Temporarily disable VPN and firewall\n* Remove customizations like 'docker-compose.*.yaml' and PHP/Apache/Nginx config while debugging.")

		output.UserOut.Printf("Resulting output will be written to:\n%s\nfile://%s\nPlease provide the file for support in Discord or the issue queue.", outputFilename, outputFilename)

		activeApps := ddevapp.GetActiveProjects()
		if len(activeApps) > 0 {
			y := util.Confirm("OK to stop running projects? This does no harm, they will be restarted")
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
		util.Success("\n\n==== Restarting previously-running DDEV projects====")
		for _, app := range activeApps {
			_ = app.Start()
		}
		util.Success("Output file written to:\n%s\nfile://%s\nPlease provide the file for support in Discord or the issue queue.", outputFilename, outputFilename)
		if testErr != nil {
			util.Failed("Failed running test_ddev.sh: %v\n. You can run it manually with `curl -sL -O https://raw.githubusercontent.com/ddev/ddev/main/cmd/ddev/cmd/scripts/test_ddev.sh && bash test_ddev.sh`", testErr)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugTestCmdCmd)
}
