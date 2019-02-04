package cmd

import (
	"github.com/drud/ddev/pkg/version"
	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugNFSMountCmd implements the ddev debug nfsmount command
var DebugNFSMountCmd = &cobra.Command{
	Use:     "nfsmount",
	Short:   "Checks to see if nfs mounting works for current project",
	Example: "ddev debug nfsmount",
	Run: func(cmd *cobra.Command, args []string) {
		testVolume := "testnfsmount"
		containerName := "testnfscontainer"

		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to debug nfsmount: %v", err)
		}
		oldContainer, err := dockerutil.FindContainerByName(containerName)
		if err == nil && oldContainer != nil {
			err = dockerutil.RemoveContainer(oldContainer.ID, 20)
			if err != nil {
				util.Failed("Failed to remove existing test container %s: %v", containerName, err)
			}
		}
		//nolint: errcheck
		dockerutil.RemoveVolume(testVolume)
		volume, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{"type": "nfs", "o": "addr=host.docker.internal,hard,nolock,rw", "device": ":" + dockerutil.MassageWindowsHostMountpoint(app.AppRoot)})
		//nolint: errcheck
		defer dockerutil.RemoveVolume(testVolume)
		if err != nil {
			util.Failed("Failed to create volume %s: %v", testVolume, err)
		}
		_ = volume
		_, _, uidStr, _ := util.GetContainerUIDGid()

		_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), containerName, []string{"sh", "-c", "findmnt -T /nfsmount && ls -d /nfsmount/.ddev"}, []string{}, []string{}, []string{"testnfsmount" + ":/nfsmount"}, uidStr, true)
		if err != nil {
			util.Failed("unable to access nfs mount: %v output=%v", err, out)
		}
		util.Success("Successfully accessed NFS mount of %s", app.AppRoot)
		output.UserOut.Printf(strings.TrimSpace(out))
	},
}

func init() {
	DebugCmd.AddCommand(DebugNFSMountCmd)
}
