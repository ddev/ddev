package cmd

import (
	"fmt"
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
	Example: `ddev auth ddev-live [token]
ddev auth ddev-live --default-org=some-ddevlive-org [token]`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			util.Failed("You must provide a DDEV API Token, e.g. 'ddev auth ddev-live [token]'. See https://dash.ddev.com/settings/integration to retrieve your your token.")
		}

		uid, _, _ := util.GetContainerUIDGid()
		c := fmt.Sprintf(`ddev-live auth --token="%s"`, args[0])
		if cmd.Flag("default-org").Changed {
			c = fmt.Sprintf(`ddev-live auth --default-org="%s" --token="%s"`, cmd.Flag("default-org").Value.String(), args[0])
		}
		_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", c}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)
		if err == nil {
			util.Success("Authentication successful!\nYou may now use the 'ddev config ddev-live' command when configuring projects.")
		} else {
			util.Failed("Failed to authenticate: %v (%v) command=%v", err, out, c)
		}
	},
}

func init() {
	AuthCmd.AddCommand(DdevLiveAuthCommand)
	DdevLiveAuthCommand.Flags().String("default-org", "", "default DDEV-Live organization")
}
