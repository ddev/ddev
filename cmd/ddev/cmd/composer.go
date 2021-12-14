package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
)

// ComposerCmd handles ddev composer
var ComposerCmd = &cobra.Command{
	Use:   "composer [command]",
	Short: "Executes a composer command within the web container",
	Long: `Executes a composer command at the project root in the web container. Generally,
any composer command can be forwarded to the container context by prepending
the command with 'ddev'.`,
	Example: `ddev composer install
ddev composer require <package>
ddev composer outdated --minor-only
ddev composer create drupal/recommended-project`,
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

		stdout, stderr, err := app.Composer(args)
		if err != nil {
			util.Failed("composer %v failed, %v. stderr=%v", args, err, stderr)
		}
		_, _ = fmt.Fprint(os.Stderr, stderr)
		_, _ = fmt.Fprint(os.Stdout, stdout)
	},
}

func init() {
	RootCmd.AddCommand(ComposerCmd)
	ComposerCmd.Flags().SetInterspersed(false)
}
