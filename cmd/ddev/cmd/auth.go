package cmd

import (
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:   "auth [command]",
	Short: "A collection of authentication commands",
	Example: `ddev auth ssh
ddev auth pantheon
ddev auth ddev-live`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)

	// Add additional auth. But they're currently in app. Should they be in global?
	// Try creating an app. If we get one, add it and add any items we find.
	// Or perhaps we should be loading completely different file.
	// How about calling it something other than "provider"? I guess the idea was "hosting provider"
	// maybe.
}
