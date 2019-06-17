package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

var (
	subdomain string
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	Use:   "share [projectname]",
	Short: "Share project on the internet via ngrok.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev share' or 'ddev share [projectname]'")
		}

		apps, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to get requested project(s): %v", err)
		}

		app := apps[0]
		if app.SiteStatus() != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		ngrokLoc, err := exec.LookPath("ngrok")
		if ngrokLoc == "" || err != nil {
			util.Failed("ngrok not found in path, please install it, see https://ngrok.com/download")
		}
		url := app.GetWebContainerDirectHTTPSURL()
		args = []string{"http"}
		if cmd.Flags().Changed("subdomain") {
			args = append(args, "--subdomain", subdomain)
		}
		if app.NgrokArgs != "" {
			args = append(args, strings.Split(app.NgrokArgs, " ")...)
		}
		args = append(args, url)
		util.Success("Running %s %s", ngrokLoc, strings.Join(args, " "))
		ngrokCmd := exec.Command(ngrokLoc, args...)
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
	DdevShareCommand.Flags().StringVarP(&subdomain, "subdomain", "S", "", "request an alternate ngrok.io subdomain (must be reserved on ngrok site)")
	RootCmd.AddCommand(DdevShareCommand)

}
