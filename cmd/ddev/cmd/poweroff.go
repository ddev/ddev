package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	docker "github.com/fsouza/go-dockerclient"
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

	// Remove any custom certs that may have been added
	_, _, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"sh", "-c", "rm -f /mnt/ddev-global-cache/custom_certs/*"}, []string{}, []string{}, []string{"ddev-global-cache" + ":/mnt/ddev-global-cache"}, "", true, false, nil)
	if err != nil {
		util.Warning("Failed removing custom certs: %v", err)
	}

	// Iterate through the list of apps built above, removing each one.
	for _, app := range apps {
		if err := app.Stop(false, false); err != nil {
			util.Failed("Failed to stop project %s: \n%v", app.GetName(), err)
		}
		util.Success("Project %s has been stopped.", app.GetName())
	}

	if err := ddevapp.RemoveSSHAgentContainer(); err != nil {
		util.Error("Failed to remove ddev-ssh-agent: %v", err)
	}
	// Remove current global network ("ddev") plus the former "ddev_default"
	removals := []string{"ddev_default"}
	for _, networkName := range removals {
		err = dockerutil.RemoveNetwork(networkName)
		_, isNoSuchNetwork := err.(*docker.NoSuchNetwork)
		// If it's a "no such network" there's no reason to report error
		if err != nil && !isNoSuchNetwork {
			util.Warning("Unable to remove network %s: %v", "ddev", err)
		}
	}
}
