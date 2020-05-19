package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/spf13/cobra"
)

// DdevLiveAuthCommand is the `ddev auth ddev-live` command
var DdevLiveAuthCommand = &cobra.Command{
	Use:   "ddev-live [token]",
	Short: "Provide a machine token for the global ddev-live auth",
	Long:  "Configure token for ddev-live authentication. See https://dash.ddev.com/settings/integration to retrieve your token.",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			util.Failed("You must provide a DDEV API Token, e.g. 'ddev auth ddev-live [token]'. See https://dash.ddev.com/settings/integration to retrieve your your token.")
		}

		uid, _, _ := util.GetContainerUIDGid()
		_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"ddev-live", "auth", "--token=" + args[0]}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)
		if err == nil {
			util.Success("Authentication successful!\nYou may now use the 'ddev config ddev-live' command when configuring projects.")
		} else {
			util.Failed("Failed to authenticate: %v (%v)", err, out)
		}
	},
}

func init() {
	AuthCmd.AddCommand(DdevLiveAuthCommand)
}
