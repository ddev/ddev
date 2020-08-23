package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
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
		powerOff()
	},
}

func init() {
	RootCmd.AddCommand(PoweroffCommand)
}

func powerOff() {
	apps, err := ddevapp.GetProjects(true)
	if err != nil {
		util.Failed("Failed to get project(s): %v", err)
	}

	// Iterate through the list of apps built above, removing each one.
	for i, app := range apps {
		if i == 0 {
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Cmd: `if [ -d /mnt/ddev-global-cache/custom_certs ]; then rm -f /mnt/ddev-global-cache/custom_certs/*; fi`,
			})
			if err == nil {
				util.Success("Removed any existing custom TLS certificates")
			} else {
				util.Warning("Could not remove existing custom TLS certificates: %v", err)
			}
		}
		if err := app.Stop(false, false); err != nil {
			util.Failed("Failed to stop project %s: \n%v", app.GetName(), err)
		}
		util.Success("Project %s has been stopped.", app.GetName())
	}

	if err := ddevapp.RemoveSSHAgentContainer(); err != nil {
		util.Error("Failed to remove ddev-ssh-agent: %v", err)
	}
}
