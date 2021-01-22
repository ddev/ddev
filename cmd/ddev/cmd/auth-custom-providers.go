package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

func init() {

	// Add additional auth. But they're currently in app. Should they be in global?
	// Try creating an app. If we get one, add it and add any items we find.
	// Or perhaps we should be loading completely different file.
	// How about calling it something other than "provider"? I guess the idea was "hosting provider"
	// maybe.
	app, err := ddevapp.NewApp("", true, "")
	if err != nil {
		util.Failed("Failed to create project: %s", err)
	}
	for k, v := range app.Providers {
		if v.AuthCommand.Command != "" {
			AuthCmd.AddCommand(&cobra.Command{
				Use:     fmt.Sprintf("%s [flags]", k),
				Short:   "Authenticate " + k + " provider",
				Long:    "Authenticate " + k + " provider",
				Example: `ddev auth ` + k,
				Run: func(cmd *cobra.Command, args []string) {
					if app.SiteStatus() != ddevapp.SiteRunning {
						util.Failed("Project %s is not running, please start it with ddev start %s", app.Name, app.Name)
					}

					util.Success("Executing command ddev auth %s", k)
					stdout, stderr, err := app.Exec(
						&ddevapp.ExecOpts{
							Service: "web",
							Cmd:     v.AuthCommand.Command,
						})
					if err == nil {
						util.Success("Authentication successful!\nYou may now use the 'ddev config %s' command when configuring this project.", k)
					} else {
						util.Failed("Failed to authenticate: %v (%v) command=%v", err, stdout+"\n"+stderr)
					}
				},
			})
		}
	}
}
