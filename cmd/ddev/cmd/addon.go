package cmd

import (
	"github.com/spf13/cobra"
)

// AddonCmd is the top-level "ddev add-on" command - effectively just a container for add-on related commands.
var AddonCmd = &cobra.Command{
	Use:     "add-on [command]",
	Aliases: []string{"addon", "add-ons", "addons"},
	Short:   "A collection of commands for managing installed 3rd party add-ons",
	Example: `ddev add-on get ddev/ddev-redis
ddev add-on remove someaddonname
ddev add-on list
ddev add-on list --installed
ddev add-on search redis
`,
}

func init() {
	RootCmd.AddCommand(AddonCmd)
}
