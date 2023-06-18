package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// PoweroffCommand contains the "ddev share" command
var PoweroffCommand = &cobra.Command{
	Use:     "poweroff",
	Short:   "Completely stop all projects and containers",
	Long:    `ddev poweroff stops all projects and containers, equivalent to ddev stop -a --stop-ssh-agent`,
	Example: `ddev poweroff`,
	Args:    cobra.NoArgs,
	Aliases: []string{"powerdown"},
	Run: func(cmd *cobra.Command, args []string) {
		ddevapp.PowerOff()
	},
}

func init() {
	RootCmd.AddCommand(PoweroffCommand)
}
