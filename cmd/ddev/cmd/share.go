package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	subdomain    string
	proxyService string
)

// DdevShareCommand contains the "ddev logs" command
var DdevShareCommand = &cobra.Command{
	Use:   "share",
	Short: "Share the project on the internet via localtunnel.",
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
		if subdomain == "" {
			subdomain = app.Name
		}

		port, _ := app.GetWebContainerPublicPort()
		p := strconv.Itoa(port)
		localhost, _ := dockerutil.GetDockerIP()
		lt, err := exec.LookPath("lt")

		if err == nil && lt != "" && (proxyService == "localtunnel" || proxyService == "") {
			args = []string{"--port", p, "--local-host", localhost, "--subdomain", subdomain, "--open"}
			util.Success("Running %s %s", lt, strings.Join(args, " "))
			cmd := exec.Command(lt, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				util.Failed("failed to run %s: %v", lt, err)
			}
			err = cmd.Wait()
			if err != nil {
				util.Failed("failed to run %s (localtunnel): %v", lt, err)
			}
			os.Exit(0)
		}

		ng, err := exec.LookPath("ngrok")
		if err == nil && ng != "" && proxyService == "ngrok" {
			url := app.GetWebContainerDirectHTTPSURL()
			args = []string{"http", url}
			util.Success("Running %s %s", ng, strings.Join(args, " "))
			cmd := exec.Command(ng, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				util.Failed("failed to run %s: %v", ng, err)
			}
			err = cmd.Wait()
			if err != nil {
				util.Failed("failed to run %s: %v", ng, err)
			}
			os.Exit(0)
		}

	},
}

func init() {
	DdevShareCommand.Flags().StringVarP(&subdomain, "subdomain", "S", "", "request an alternate subdomain")
	DdevShareCommand.Flags().StringVarP(&proxyService, "proxy-service", "P", "", "choose a proxy service (localtunnel or ngrok)")
	RootCmd.AddCommand(DdevShareCommand)

}
