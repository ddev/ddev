package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/nodeps"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
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

		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Dir:     "/var/www/html",
			Cmd:     fmt.Sprintf("composer %s", strings.Join(args, " ")),
		})
		if err != nil {
			util.Failed("composer command failed: %v", err)
		}
		if runtime.GOOS == "windows" && !nodeps.IsDockerToolbox() {
			replaceSimulatedLinks(app.AppRoot)
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

// replaceSimulatedLinks() walks the path provided and tries to replace XSym links with real ones.
func replaceSimulatedLinks(path string) {
	links, err := fileutil.FindSimulatedXsymSymlinks(path)
	if err != nil {
		util.Warning("Error finding XSym Symlinks: %v", err)
	}
	if len(links) == 0 {
		return
	}

	if !fileutil.CanCreateSymlinks() {
		util.Warning("This host computer is unable to create real symlinks, please see the docs to enable developer mode:\n%s\nNote that the simulated symlinks created inside the container will work fine for most projects.", "https://ddev.readthedocs.io/en/latest/users/developer-tools/#windows-os-and-ddev-composer")
		return
	}

	err = fileutil.ReplaceSimulatedXsymSymlinks(links)
	if err != nil {
		util.Warning("Failed replacing simulated symlinks: %v", err)
	}
	replacedLinks := make([]string, 0)
	for _, l := range links {
		replacedLinks = append(replacedLinks, l.LinkLocation)
	}
	util.Success("Replaced these simulated symlinks with real symlinks: %v", replacedLinks)
	return
}
