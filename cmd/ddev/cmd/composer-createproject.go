package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
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
		// If the user hasn't passed any positional arguments, we're not actually executing an
		// install and nothing will be deleted -- for example, 'ddev composer create-project -h'
		removeData := false
		if len(positionals) > 0 {
			util.Warning("Warning: Any existing contents of the project root will be overwritten")
			removeData = true
			if !util.Confirm("Would you like to continue?") {
				util.Failed("create-project cancelled")
			}
		}

		// Only remove data if we expect an install
		composerTmpDir := "/tmp/composer"
		if removeData {
			// The install directory may be populated if a previous create-project was
			// executed using the same container.
			output.UserOut.Printf("Ensuring composer install directory is empty")
			_, _, _ = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"sh", "-c", fmt.Sprintf("rm -rf %s", composerTmpDir)},
			})

			// Remove any contents of project root
			util.Warning("Removing any existing files in project root")
			_, _, _ = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"sh", "-c", "rm -rf /var/www/html/*"},
			})
		}

		// Build container composer command
		installDir := path.Join(composerTmpDir, dir)
		composerCmd := append([]string{}, "composer", "create-project")
		composerCmd = append(composerCmd, options...)
		composerCmd = append(composerCmd, pack, installDir, ver)

		output.UserOut.Printf("Executing composer command: %s\n", strings.Join(composerCmd, " "))
		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     composerCmd,
		})
		if err != nil {
			util.Failed("Failed to execute create-project command")
		}

		if len(stdout) > 0 {
			fmt.Println(stdout)
		}

		// Move project to app root if an install is underway
		if removeData {
			output.UserOut.Printf("Moving installation to project root")
			_, _, _ = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     []string{"sh", "-c", fmt.Sprintf("if [ -d %s ]; then mv %s /var/www/html/; fi", composerTmpDir, path.Join(composerTmpDir, "*"))},
			})
		}
	},
}

func init() {
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCreateProjectCmd.DisableFlagParsing = true
}
