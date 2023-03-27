package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	Use:   "share [project]",
	Short: "Share project on the internet via ngrok.",
	Long:  `Requires a free or paid account on ngrok.com, use the "ngrok config add-authtoken <token>" command to set up ngrok. Although a few ngrok commands are supported directly, any ngrok flag can be added in the ngrok_args section of .ddev/config.yaml.`,
	Example: `ddev share
ddev share --basic-auth username:pass1234
ddev share myproject`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev share' or 'ddev share [projectname]'")
		}
		apps, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		app := apps[0]

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		ngrokLoc, err := exec.LookPath("ngrok")
		if ngrokLoc == "" || err != nil {
			util.Failed("ngrok not found in path, please install it, see https://ngrok.com/download")
		}
		urls := []string{app.GetWebContainerDirectHTTPURL()}

		var ngrokErr error
		for _, url := range urls {
			ngrokArgs := []string{"http"}
			ngrokArgs = append(ngrokArgs, url)
			if app.NgrokArgs != "" {
				ngrokArgs = append(ngrokArgs, strings.Split(app.NgrokArgs, " ")...)
			}
			if cmd.Flags().Changed("basic-auth") {
				auth, err := cmd.Flags().GetString("basic-auth")
				if err != nil {
					util.Failed("unable to get --basic-auth flag: %v", err)
				}
				ngrokArgs = append(ngrokArgs, "--basic-auth="+auth)
			}
			if cmd.Flags().Changed("subdomain") {
				sub, err := cmd.Flags().GetString("subdomain")
				if err != nil {
					util.Failed("unable to get --subdomain flag: %v", err)
				}
				ngrokArgs = append(ngrokArgs, "--subdomain="+sub)
			}

			ngrokCmd := exec.Command(ngrokLoc, ngrokArgs...)
			ngrokCmd.Stdout = os.Stdout
			ngrokCmd.Stderr = os.Stderr
			err = ngrokCmd.Start()
			if err != nil {
				util.Failed("Failed to run %s %s: %v", ngrokLoc, strings.Join(ngrokArgs, " "), err)
			}
			util.Success("Running %s %s", ngrokLoc, strings.Join(ngrokArgs, " "))

			ngrokErr = ngrokCmd.Wait()
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
				util.Error("ngrok exited: %v", exitErr)
				break
			}
			// Otherwise we'll continue and do the next url or exit
		}
		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().String("basic-auth", "", `works as in "ngrok http --basic-auth username:pass1234"`)
	DdevShareCommand.Flags().String("subdomain", "", `requires a paid ngrok account, works as in "ngrok http --subdomain mysite"`)
}
