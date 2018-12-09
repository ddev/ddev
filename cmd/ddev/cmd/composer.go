package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var ComposerCmd = &cobra.Command{
	Use:   "composer [command]",
	Short: "Executes a composer command within the web container",
	Long: `Executes a composer command at the project root in the web container. Generally,
any composer command can be forwarded to the container context by prepending
the command with 'ddev'. For example:

ddev composer require <package>
ddev composer outdated --minor-only`,
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		if app.SiteStatus() != ddevapp.SiteRunning {
			if err = app.Start(); err != nil {
				util.Failed("Failed to start %s: %v", app.Name, err)
			}
		}

		output.UserOut.Printf("Executing [composer %s] at the project root (/var/www/html in the container, %s on the host)", strings.Join(args, " "), app.AppRoot)
		stdout, _, _ := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Dir:     "/var/www/html",
			Cmd:     append([]string{"composer"}, args...),
		})
		if runtime.GOOS == "windows" && !util.IsDockerToolbox() {
			if fileutil.CanCreateSymlinks() {
				links, err := fileutil.FindSimulatedXsymSymlinks(app.AppRoot)
				if err == nil {
					err = fileutil.ReplaceSimulatedXsymSymlinks(links)
					if err != nil {
						util.Warning("Failed replacing simulated symlinks: %v", err)
					}
					util.Success("Replaced simulated symlinks with real symlinks: %v", links)
				} else {
					util.Warning("Error finding simulated symlinks")
				}
			} else {
				util.Warning("This host computer is unable to create symlinks, please see the docs to enable developer mode.")
			}
		}

		if len(stdout) > 0 {
			fmt.Println(stdout)
		}
	},
}

func init() {
	RootCmd.AddCommand(ComposerCmd)
	ComposerCmd.Flags().SetInterspersed(false)
}
