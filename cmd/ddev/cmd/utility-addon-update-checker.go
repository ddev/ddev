package cmd

import (
	"errors"
	"io"
	"net/http"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var addonUpdateCheckerURLs = []string{
	"https://ddev.com/s/addon-update-checker.sh",
	"https://raw.githubusercontent.com/ddev/ddev-addon-template/main/.github/scripts/update-checker.sh",
}

// AddonUpdateCheckerCmd implements the "ddev utility addon-update-checker" command
var AddonUpdateCheckerCmd = &cobra.Command{
	Use:   "addon-update-checker",
	Args:  cobra.NoArgs,
	Short: "Run the DDEV add-on update checker script (for add-on developers)",
	Long: `Fetch and run the DDEV add-on update checker script from https://ddev.com/s/addon-update-checker.sh
This is a tool for add-on developers to verify their add-on's scripts and tooling are up to date.

If the target directory contains install.yaml, the checker runs there. Otherwise, it scans immediate
subdirectories for install.yaml and runs the checker in each one, which is useful when working in a
workspace with multiple add-ons checked out alongside each other.`,
	Example: `# Run in current directory (must contain install.yaml, or subdirs must)
ddev utility addon-update-checker

# Run in a specific add-on directory
ddev ut addon-update-checker -d /path/to/my-addon

# Run across all add-ons in a workspace
ddev ut addon-update-checker -d /path/to/my-addons-workspace
`,
	Run: func(cmd *cobra.Command, args []string) {
		bashPath := util.FindBashPath()
		client := &http.Client{Timeout: 30 * time.Second}

		var script string
		for i, url := range addonUpdateCheckerURLs {
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				if i < len(addonUpdateCheckerURLs)-1 {
					util.Warning("Unable to fetch update checker script from %s, trying fallback", url)
					continue
				}
				util.Failed("Unable to fetch update checker script: %v", err)
			}
			b, err := io.ReadAll(resp.Body)
			util.CheckClose(resp.Body)
			if err != nil {
				util.Failed("Unable to read update checker script: %v", err)
			}
			script = string(b)
			break
		}

		rootDir := cmd.Flag("dir").Value.String()
		if rootDir == "" {
			var err error
			rootDir, err = os.Getwd()
			if err != nil {
				util.Failed("Unable to determine current directory: %v", err)
			}
		}

		hostCmd := exec.HostCommand(bashPath, "-s")
		hostCmd.Stdin = strings.NewReader(script)
		hostCmd.Stdout = os.Stdout
		hostCmd.Stderr = os.Stderr
		hostCmd.Dir = rootDir

		if err := hostCmd.Run(); err != nil {
			var exiterr *osexec.ExitError
			if errors.As(err, &exiterr) {
				output.UserErr.Exit(exiterr.ExitCode())
			}
			util.Failed("Unable to run update checker script: %v", err)
		}
	},
}

func init() {
	AddonUpdateCheckerCmd.Flags().StringP("dir", "d", "", "Directory of the add-on to check, or a workspace containing multiple add-on directories (defaults to current directory)")
	DebugCmd.AddCommand(AddonUpdateCheckerCmd)
}
