package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/spf13/cobra"
	"path/filepath"
	"runtime"
	"strings"
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

		hostDockerInternal, err := dockerutil.GetHostDockerInternalIP()
		if err != nil {
			util.Failed("failed to GetHostDockerInternalIP(): %v", err)
		}
		if hostDockerInternal == "" {
			hostDockerInternal = "host.docker.internal"
		}
		shareDir := app.AppRoot
		// Workaround for Catalina sharing nfs as /System/Volumes/Data
		if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", app.AppRoot)) {
			shareDir = filepath.Join("/System/Volumes/Data", app.AppRoot)
		}
		volume, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{"type": "nfs", "o": fmt.Sprintf("addr=%s,hard,nolock,rw", hostDockerInternal), "device": ":" + dockerutil.MassageWindowsNFSMount(shareDir)})
		//nolint: errcheck
		defer dockerutil.RemoveVolume(testVolume)
		if err != nil {
			util.Failed("Failed to create volume %s: %v", testVolume, err)
		}
		_ = volume
		uidStr, _, _ := util.GetContainerUIDGid()

		_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), containerName, []string{"sh", "-c", "findmnt -T /nfsmount && ls -d /nfsmount/.ddev"}, []string{}, []string{}, []string{"testnfsmount" + ":/nfsmount"}, uidStr, true, false, nil)
		if err != nil {
			util.Warning("NFS does not seem to be set up yet, see debugging instructions at https://ddev.readthedocs.io/en/stable/users/performance/#debugging-ddev-start-failures-with-nfs_mount_enabled-true")
			util.Failed("Details: error=%v\noutput=%v", err, out)
		}
		output.UserOut.Printf(strings.TrimSpace(out))
		util.Success("")
		util.Success("Successfully accessed NFS mount of %s", app.AppRoot)
		switch {
		case app.NFSMountEnabledGlobal:
			util.Success("nfs_mount_enabled is set globally")
		case app.NFSMountEnabled:
			util.Success("nfs_mount_enabled is true in this project (%s), but is not set globally", app.Name)
		default:
			util.Warning("nfs_mount_enabled is not set either globally or in this project. \nUse `ddev config global --nfs-mount-enabled` to enable it.")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugNFSMountCmd)
}
