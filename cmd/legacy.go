package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

const netName = "drud_default"

// LegacyCmd represents the local command
var LegacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Local dev for legacy sites.",
	Long:  `Manage your local Legacy development environment with these commands.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		parseLegacyArgs(args)
	},
}

func init() {
	RootCmd.AddCommand(LegacyCmd)
}

func parseLegacyArgs(args []string) {
	activeApp = cfg.ActiveApp
	activeDeploy = cfg.ActiveDeploy

	if len(args) > 1 {
		if args[0] != "" {
			activeApp = args[0]
		}

		if args[1] != "" {
			activeDeploy = args[1]
		}
	}
	if activeApp == "" {
		log.Fatalln("No app name found. app_name and deploy_name are expected as arguments.")
	}
	if activeDeploy == "" {
		log.Fatalln("No deploy name found. app_name and deploy_name are expected as arguments.")
	}
	if posString([]string{"default", "staging", "production"}, activeDeploy) == -1 {
		log.Fatalln("Bad environment name.")
	}
}
