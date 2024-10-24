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

// CdCmd is the top-level "ddev cd" command
var CdCmd = &cobra.Command{
	Use:   "cd [project-name]",
	Short: "Uses shell built-in 'cd' to change to a project directory",
	Long: fmt.Sprintf(`To enable 'ddev cd' command, source the ddev.sh script from your rc-script.

From bash:

printf '\nsource "%s"' >> ~/.bashrc

From zsh:

printf '\nsource "%s"' >> ~/.zshrc

From fish:

echo \n'source "%s"' >> ~/.config/fish/config.fish

Restart your shell, and use 'ddev cd project-name'.
`, filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh"), filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh"), filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish")),
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Example: `ddev cd
command ddev cd project-name
ddev cd project-name`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, file := range []string{"ddev.sh", "ddev.fish"} {
			if !fileutil.FileExists(filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells", file)) {
				util.Failed("Unable to find %s in %s", file, globalconfig.GetGlobalDdevDir())
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
