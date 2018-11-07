package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"

	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var ComposerCmd = &cobra.Command{
	Use: "composer [command]",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		stdout, _, _ := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
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
