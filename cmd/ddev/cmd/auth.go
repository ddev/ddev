package cmd

import (
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:     "auth [command]",
	Short:   "A collection of authentication commands",
	Example: `ddev auth ssh`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)

	// Add hidden pantheon subcommand for people who have it in their fingers
	AuthCmd.AddCommand(&cobra.Command{
		Use:    "pantheon",
		Short:  "ddev auth pantheon is no longer needed, see docs",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			output.UserOut.Print("`ddev auth pantheon` is no longer needed, see docs")
		},
	})

}
