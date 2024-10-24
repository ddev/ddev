package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// CdCmd is the top-level "ddev cd" command
var CdCmd = &cobra.Command{
	Use:   "cd [project-name]",
	Short: "Uses shell built-in 'cd' to change to a project directory",
	Long: `For bash/zsh add this function to your ~/.bashrc or ~/.zshrc,
then restart your shell, and use 'ddev cd project-name':

ddev() {
  if [ "$1" = "cd" ] && [ -n "$2" ]; then
    cd "$(DDEV_VERBOSE=false command ddev cd "$2")"
  else
    command ddev "$@"
  fi
}

For fish add this function to your ~/.config/fish/config.fish,
then restart your shell, and use 'ddev cd project-name':

function ddev
  if test (count $argv) -eq 2 -a "$argv[1]" = "cd"
    cd "$(DDEV_VERBOSE=false command ddev cd $argv[2])"
  else
    command ddev $argv
  end
end`,
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Example: `ddev cd
command ddev cd project-name
ddev cd project-name`,
	Run: func(cmd *cobra.Command, args []string) {
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
