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
	Long:  `Use "ddev share" or add on extra ngrok commands, like "ddev share --subdomain my-reserved-subdomain". Although a few ngrok commands are supported directly, any ngrok flag can be added in the ngrok_args section of .ddev/config.yaml. You will need to create an account on ngrok.com and use the "ngrok authtoken" command to set up ngrok.`,
	Example: `ddev share
ddev share --subdomain some-subdomain
ddev share --auth authkey`,
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
		url := app.GetWebContainerDirectHTTPSURL()
		useHTTP, err := cmd.Flags().GetBool("use-http")
		if err != nil {
			util.Failed("failed to get use-http flag: %v", err)
		}

		if useHTTP {
			url = app.GetWebContainerDirectHTTPURL()
		}
		ngrokArgs := []string{"http"}
		if app.NgrokArgs != "" {
			ngrokArgs = append(ngrokArgs, strings.Split(app.NgrokArgs, " ")...)
		}
		ngrokArgs = append(ngrokArgs, url)
		x := cmd.Flags().Args()
		_ = x
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
	DdevShareCommand.Flags().String("subdomain", "", `ngrok --subdomain argument, as in "ngrok --subdomain my-subdomain"`)
	DdevShareCommand.Flags().Bool("use-http", false, `Set to true to use unencrypted http local tunnel (required if you do not have an ngrok.com account)"`)

	DdevShareCommand.Flags().Bool("inspect", false, `ngrok --inspect argument, as in "ngrok --inspect=true"`)

	DdevShareCommand.Flags().String("auth", "", `ngrok --auth flag, as in --auth "user:pass"`)
}
