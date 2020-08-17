package cmd

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/spf13/cobra"
)

// PantheonAuthCommand is the `ddev auth pantheon` command
var PantheonAuthCommand = &cobra.Command{
	Use:     "pantheon [token]",
	Short:   "Provide a machine token for the global pantheon auth",
	Long:    "Configure global machine token for pantheon authentication. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.",
	Example: `ddev auth pantheon`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			util.Failed("You must provide a Pantheon machine token, e.g. 'ddev auth pantheon [token]'. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.")
		}
		if len(args) != 1 {
			util.Failed("Too many arguments detected. Please provide only your Pantheon Machine token., e.g. 'ddev auth pantheon [token]'. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.")
		}

		uid, _, _ := util.GetContainerUIDGid()
		_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"terminus", "auth:login", "--machine-token=" + args[0]}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)
		if err == nil {
			util.Success("Authentication successful!\nYou may now use the 'ddev config pantheon' command when configuring projects.")
		} else {
			util.Failed("Failed to authenticate: %v (%v)", err, out)
		}
	},
}

func init() {
	AuthCmd.AddCommand(PantheonAuthCommand)
}
