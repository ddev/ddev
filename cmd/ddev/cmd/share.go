package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	Use:   "share",
	Short: "Share project on the internet via ngrok.",
	Long:  `Use "ddev share" or add on extra ngrok commands, like "ddev share --subdomain some-subdomain". Although a few ngrok commands are supported directly, any ngrok flag can be added in the ngrok_args section of .ddev/config.yaml. You will want to create an account on ngrok.com and use the "ngrok authtoken" command to set up ngrok.`,
	Example: `ddev share
ddev share --subdomain some-subdomain
ddev share --use-http`,
	//Args: cobra.ArbitraryArgs,
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
		urls := []string{app.GetWebContainerDirectHTTPSURL(), app.GetWebContainerDirectHTTPURL()}

		// If they provided the --use-http flag, we'll not try both https and http
		useHTTP, err := cmd.Flags().GetBool("use-http")
		if err != nil {
			util.Failed("failed to get use-http flag: %v", err)
		}

		if useHTTP {
			urls = []string{app.GetWebContainerDirectHTTPURL()}
		}

		var ngrokErr error
		for _, url := range urls {
			ngrokArgs := []string{"http"}
			if app.NgrokArgs != "" {
				ngrokArgs = append(ngrokArgs, strings.Split(app.NgrokArgs, " ")...)
			}
			ngrokArgs = append(ngrokArgs, url)
			ngrokArgs = append(ngrokArgs, args...)

			if strings.Contains(url, "http://") {
				util.Warning("Using local http URL, your data may be exposed on the internet. Create a free ngrok account instead...")
				time.Sleep(time.Second * 3)
			}
			util.Success("Running %s %s", ngrokLoc, strings.Join(ngrokArgs, " "))
			ngrokCmd := exec.Command(ngrokLoc, ngrokArgs...)
			ngrokCmd.Stdout = os.Stdout
			ngrokCmd.Stderr = os.Stderr
			ngrokErr = ngrokCmd.Run()
			// nil result means ngrok ran and exited normally.
			// It seems to do this fine when hit by SIGTERM or SIGINT
			if ngrokErr == nil {
				break
			}

			exitErr, ok := ngrokErr.(*exec.ExitError)
			if !ok {
				// Normally we'd have an ExitError, but if not, notify
				util.Error("ngrok exited: %v", ngrokErr)
				break
			}

			exitCode := exitErr.ExitCode()
			// In the case of exitCode==1, ngrok seems to have died due to an error,
			// most likely inadequate user permissions.
			if exitCode != 1 {
				util.Warning("ngrok exited: %v", exitErr)
				break
			}
			// Otherwise we'll continue and do the next url or exit
			util.Warning("ngrok exited: %v", exitErr)
		}
		util.Warning("ngrokErr: %v, goprocs: %v", ngrokErr, runtime.NumGoroutine())
		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().String("subdomain", "", `ngrok --subdomain argument, as in "ngrok --subdomain my-subdomain:, requires paid ngrok.com account"`)
	DdevShareCommand.Flags().Bool("use-http", false, `Set to true to use unencrypted http local tunnel (required if you do not have an ngrok.com account)"`)
}
