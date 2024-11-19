package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/heredoc"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"path/filepath"
	"strings"
)

var (
	bashFile = filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.sh")
	fishFile = filepath.Join(globalconfig.GetGlobalDdevDir(), "commands/host/shells/ddev.fish")
)

// CdCmd is the top-level "ddev cd" command
var CdCmd = &cobra.Command{
	Use:   "cd [project-name]",
	Short: "Uses shell built-in 'cd' to change to a project directory",
	Long: heredoc.Doc(fmt.Sprintf(`
		To enable the 'ddevcd' function, source the ddev.sh script from your rc-script.

		For bash:

		printf '\n[ -f "%s" ] && source "%s"\n' >> ~/.bashrc

		For zsh:

		printf '\n[ -f "%s" ] && source "%s"\n' >> ~/.zshrc

		For fish:

		printf '\n[ -f "%s" ] && source "%s"\n' >> ~/.config/fish/config.fish

		Restart your shell, and use 'ddevcd project-name'.
		`, bashFile, bashFile, bashFile, bashFile, fishFile, fishFile)),
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Example: heredoc.DocI2S(`
		ddev cd
		ddevcd project-name
	`),
	Run: func(cmd *cobra.Command, args []string) {
		for _, file := range []string{bashFile, fishFile} {
			if !fileutil.FileExists(file) {
				util.Failed("Unable to find file: %s", file)
			}
		}
		if cmd.Flags().Changed("list") {
			if len(args) != 0 {
				util.Failed("The provided flag does not take any arguments")
			}
			projects, _ := cmd.ValidArgsFunction(cmd, args, "")
			output.UserOut.Println(strings.Join(projects, "\n"))
			return
		}
		if cmd.Flags().Changed("get-approot") {
			if len(args) != 1 {
				util.Failed("This command only takes one argument: project-name")
			}
			projectName := args[0]
			app, err := ddevapp.GetActiveApp(projectName)
			if err != nil {
				util.Failed("Failed to find path for project: %v", err)
			}
			output.UserOut.Println(app.AppRoot)
			return
		}
		output.UserOut.Println(cmd.Long)
	},
}

func init() {
	CdCmd.Flags().BoolP("get-approot", "", false, "Get the full path to the project root directory")
	_ = CdCmd.Flags().MarkHidden("get-approot")
	CdCmd.Flags().BoolP("list", "", false, "Get project names")
	_ = CdCmd.Flags().MarkHidden("list")
	RootCmd.AddCommand(CdCmd)
}
