package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:   "auth [command]",
	Short: "A collection of authentication commands",
	Example: `ddev auth ssh
ddev auth pantheon
ddev auth ddev-live`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)

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
			// since for-loop k,v are references, we need to get a copy of them.
			kl := k
			vl := v
			AuthCmd.AddCommand(&cobra.Command{
				Use:     fmt.Sprintf("%s [flags]", kl),
				Short:   "Authenticate " + kl + " provider",
				Long:    "Authenticate " + kl + " provider",
				Example: `ddev auth ` + kl,
				Run: func(cmd *cobra.Command, args []string) {
					if app.SiteStatus() != ddevapp.SiteRunning {
						util.Failed("Project %s is not running, please start it with ddev start %s", app.Name, app.Name)
					}

					util.Success("Executing command ddev auth %s", kl)
					stdout, stderr, err := app.Exec(
						&ddevapp.ExecOpts{
							Service: "web",
							Cmd:     vl.AuthCommand.Command,
						})
					if err == nil {
						util.Success("Authentication successful!\nYou may now use the 'ddev config %s' command when configuring this project.", kl)
					} else {
						util.Failed("Failed to authenticate: %vl (%vl) command=%vl", err, stdout+"\n"+stderr)
					}
				},
			})
		}
	}
}
