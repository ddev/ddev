package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevXHGuiCmd represents the xhgui command.
var DdevXHGuiCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("active", 1),
	Use:               "xhgui [on|off|launch|status]",
	Short:             "Starts or checks status of XHGui performance monitoring and UI",
	Example: `ddev xhgui
ddev xhgui launch
ddev xhgui on
ddev xhgui off
ddev xhgui status
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			return fmt.Errorf("unable to get project: %v", err)
		}

		app.DockerEnv()

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		if app.GetXHProfMode() != types.XHProfModeXHGui {
			util.Failed("XHProf Mode is set to '%s', can't use 'ddev xhgui'. Use 'ddev config global --xhprof-mode=xhgui' to enable.", app.GetXHProfMode())
		}

		action := "launch"
		if len(args) == 1 {
			action = args[0]
		}

		switch action {

		case "launch":
			if !ddevapp.XHGuiStatus(app) {
				err = ddevapp.XHGuiSetup(app)
				if err != nil {
					return err
				}
				util.Success("Enabled XHProf for XHGui.")
			}
			if !output.JSONOutput {
				out, err := exec.RunHostCommand("ddev", "launch", app.GetXHGuiURL())
				if err != nil {
					util.Warning("failed to launch xhgui: output='%s', err=%v", out, err)
				}
			}

		case "on", "enable", "start", "true":
			if err = ddevapp.XHGuiSetup(app); err != nil {
				return err
			}

		case "false", "off", "disable", "stop":
			if err = ddevapp.XHProfDisable(app); err != nil {
				return err
			}

		case "status":
			var status bool
			if status, err = ddevapp.XHProfStatus(app); err != nil {
				return err
			}
			switch status {
			case true:
				util.Success("XHProf is enabled and capturing performance information.")
			case false:
				util.Success("XHProf is disabled.")
			}

			status = ddevapp.XHGuiStatus(app)
			switch status {
			case true:
				util.Success("The XHGui service is running and you can access it at %s", app.GetXHGuiURL())
			case false:
				util.Success("XHGui is disabled.")
			}

		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(DdevXHGuiCmd)
}
