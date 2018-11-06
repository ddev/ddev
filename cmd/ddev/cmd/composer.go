package cmd

import (
	"os"

	"github.com/drud/ddev/pkg/ddevapp"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var ComposerCmd = &cobra.Command{
	Use: "composer [command]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			return
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     append([]string{"composer"}, args...),
		})
		if err != nil {
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(ComposerCmd)
}
