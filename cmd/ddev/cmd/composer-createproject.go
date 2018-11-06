package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var ComposerCreateProjectCmd = &cobra.Command{
	Use:   "create-project [options] [package] [directory] [version]",
	Short: "passes arguments to 'composer create-project' within the container",
	Long:  `'ddev composer create-project' will direct the use of 'composer create-project' within the context of the container. Projects will be installed to a temporary directory and moved to the project root directory after installation.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse any positional arguments
		positionals := make([]string, 0)
		options := make([]string, 0)
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				options = append(options, arg)
				continue
			}

			positionals = append(positionals, arg)
		}

		var pack, dir, ver string
		if len(positionals) >= 3 {
			pack = positionals[len(positionals)-3]
			dir = positionals[len(positionals)-2]
			ver = positionals[len(positionals)-1]
		} else if len(positionals) == 2 {
			pack = positionals[len(positionals)-2]
			dir = positionals[len(positionals)-1]
		} else if len(positionals) == 1 {
			pack = positionals[len(positionals)-1]
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		// Ensure project is running
		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		// Make the user confirm that existing contents will be deleted
		util.Warning("Warning: Existing contents of the project root will be overwritten")
		for {
			r := strings.ToLower(util.Prompt("Would you like to continue? [Y/n]", "yes"))
			if r == "no" || r == "n" {
				util.Failed("create-project cancelled")
			}

			if r == "yes" || r == "y" {
				break
			}
		}

		// Build container composer command
		installDir := path.Join("/tmp/composer", dir)
		composerCmd := append([]string{}, "composer", "create-project")
		composerCmd = append(composerCmd, options...)
		composerCmd = append(composerCmd, pack, installDir, ver)

		fmt.Println(composerCmd)
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     composerCmd,
		})
		if err != nil {
			util.Failed("Failed to execute create-project command")
		}

		// Move project to app root
		_, _, _ = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     []string{"sh", "-c", "if [ -d /tmp/composer ]; then mv /tmp/composer/* /var/www/html/; fi"},
		})
	},
}

func init() {
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCreateProjectCmd.DisableFlagParsing = true
	ComposerCreateProjectCmd.Flags().SetInterspersed(false)
}
