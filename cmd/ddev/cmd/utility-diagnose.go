package cmd

import (
	"os"
	"path"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DiagnoseCmd implements the ddev utility diagnose command
var DiagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose common DDEV issues with concise, actionable output",
	Long: `Run quick diagnostics on your DDEV installation and current project.
This command checks:
- Docker environment and connectivity
- Network configuration
- HTTPS/mkcert setup
- Current project health (if in a project directory)

For comprehensive output suitable for issue reports, use 'ddev debug test' instead.`,
	Example: `ddev utility diagnose
ddev ut diagnose
DDEV_DIAGNOSE_FULL=true ddev utility diagnose  # Include test project creation`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		bashPath := util.FindBashPath()
		tmpDir := os.TempDir()

		// Copy embedded script to temp directory
		err := fileutil.CopyEmbedAssets(bundledAssets, "scripts", tmpDir, nil)
		if err != nil {
			util.Failed("Failed to copy diagnose_ddev.sh to %s: %v", tmpDir, err)
		}

		scriptPath := tmpDir
		if nodeps.IsWindows() {
			if !fileutil.FileExists(bashPath) {
				util.Failed("%s does not exist, please install git-bash to use 'ddev utility diagnose'", bashPath)
			}
			scriptPath = util.WindowsPathToCygwinPath(tmpDir)
		}

		c := []string{"-c", path.Join(scriptPath, "diagnose_ddev.sh")}
		util.Debug("Running %s %v", bashPath, c)

		// Show introductory message
		output.UserOut.Println("Running DDEV diagnostics...")
		output.UserOut.Println()
		util.Success("Quick troubleshooting tips:")
		util.Success("* Use latest stable DDEV version")
		util.Success("* ddev poweroff && ddev start")
		util.Success("* Restart Docker Provider")
		util.Success("* Temporarily disable VPN and firewall")
		util.Success("* Troubleshooting guide: https://docs.ddev.com/en/stable/users/usage/troubleshooting/")
		output.UserOut.Println()
		output.UserOut.Println("For comprehensive output for issue reports, use 'ddev debug test' instead.")
		output.UserOut.Println()

		// Run the script with output to stdout
		err = exec.RunInteractiveCommandWithOutput(bashPath, c, os.Stdout)
		if err != nil {
			// Script already printed diagnostic info, just exit with error code
			os.Exit(1)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DiagnoseCmd)
}
