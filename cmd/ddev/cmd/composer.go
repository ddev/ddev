package cmd

import (
	"fmt"
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

		if len(stdout) > 0 {
			fmt.Println(stdout)
		}
	},
}

func init() {
	RootCmd.AddCommand(ComposerCmd)
	ComposerCmd.Flags().SetInterspersed(false)
}
