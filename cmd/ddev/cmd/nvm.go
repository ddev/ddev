package cmd

import (
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// NvmCmd implements the ddev nvm command
var NvmCmd = &cobra.Command{
	Use:   "nvm [args]",
	Short: "nvm has been removed from DDEV",
	Long: `The 'ddev nvm' command has been removed in DDEV v1.25.0.

Use the 'nodejs_version' configuration in .ddev/config.yaml instead:

  nodejs_version: "22"  # or any version like "20", "18.16.0", etc.

After changing nodejs_version, restart your project with 'ddev restart'.

For more information, see:
https://docs.ddev.com/en/stable/users/configuration/config/#nodejs_version

If you need to use nvm directly, you can still run it inside the container:
  ddev ssh
  nvm install 20
`,
	Run: func(_ *cobra.Command, args []string) {
		output.UserErr.Print(`The 'ddev nvm' command has been removed in DDEV v1.25.0.

Use the 'nodejs_version' configuration in .ddev/config.yaml instead:

  nodejs_version: "22"  # or any version like "20", "18.16.0", etc.

After changing nodejs_version, restart your project with 'ddev restart'.

For more information, see:
https://docs.ddev.com/en/stable/users/configuration/config/#nodejs_version

`)
		util.Failed("ddev nvm has been removed")
	},
}

func init() {
	RootCmd.AddCommand(NvmCmd)
}
