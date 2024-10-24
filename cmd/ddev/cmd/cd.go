package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"path/filepath"
)

var bashFile = filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh")
var fishFile = filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish")

// CdCmd is the top-level "ddev cd" command
var CdCmd = &cobra.Command{
	Use:   "cd [project-name]",
	Short: "Uses shell built-in 'cd' to change to a project directory",
	Long: fmt.Sprintf(`To enable 'ddev cd' command, source the ddev.sh script from your rc-script.

From bash:

printf '\n[ -f "%s" ] && source "%s"' >> ~/.bashrc

From zsh:

printf '\n[ -f "%s" ] && source "%s"' >> ~/.zshrc

From fish:

echo \n'[ -f "%s" ] && source "%s"' >> ~/.config/fish/config.fish

Restart your shell, and use 'ddev cd project-name'.
`, bashFile, bashFile, bashFile, bashFile, fishFile, fishFile),
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Example: `ddev cd
command ddev cd project-name
ddev cd project-name`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, file := range []string{bashFile, fishFile} {
			if !fileutil.FileExists(file) {
				util.Failed("Unable to find file: %s", file)
			}

		}
		if len(args) < 1 {
			output.UserOut.Println(cmd.Long)
			return
		}
		if len(args) != 1 {
			util.Failed("This command only takes one argument: project-name")
		}
		projectName := args[0]
		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to find path for project: %v", err)
		}
		output.UserOut.Println(app.AppRoot)
	},
}

func init() {
	RootCmd.AddCommand(CdCmd)
}