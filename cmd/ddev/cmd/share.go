package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	Use:   "share",
	Short: "Share project on the internet via ngrok.",
	Long:  `Use "ddev share" or add on extra ngrok commands, like "ddev share --subdomain my-reserved-subdomain"`,
	Example: `ddev share
ddev share --subdomain some-subdomain
ddev share --auth authkey`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to get requested project: %v", err)
		}
		if app.SiteStatus() != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		ngrokLoc, err := exec.LookPath("ngrok")
		if ngrokLoc == "" || err != nil {
			util.Failed("ngrok not found in path, please install it, see https://ngrok.com/download")
		}
		url := app.GetWebContainerDirectHTTPSURL()
		ngrokArgs := []string{"http"}
		if app.NgrokArgs != "" {
			ngrokArgs = append(ngrokArgs, strings.Split(app.NgrokArgs, " ")...)
		}
		ngrokArgs = append(ngrokArgs, url)
		ngrokArgs = append(ngrokArgs, args...)
		util.Success("Running %s %s", ngrokLoc, strings.Join(ngrokArgs, " "))
		ngrokCmd := exec.Command(ngrokLoc, ngrokArgs...)
		ngrokCmd.Stdout = os.Stdout
		ngrokCmd.Stderr = os.Stderr
		err = ngrokCmd.Start()
		if err != nil {
			util.Failed("failed to run %s: %v", ngrokLoc, err)
		}
		err = ngrokCmd.Wait()
		if err != nil {
			util.Failed("failed to run %s: %v", ngrokLoc, err)
		}
		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().SetInterspersed(false)
}
