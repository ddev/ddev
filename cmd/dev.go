package cmd

import (
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/bootstrap/cli/local"

	"github.com/spf13/cobra"
)

var (
	serviceType string
)

const netName = "drud_default"

// LocalDevCmd represents the local command
var LocalDevCmd = &cobra.Command{
	Use:     "dev",
	Aliases: []string{"legacy"},
	Short:   "Local dev for sites.",
	Long:    `Manage your local development environment with these commands.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !strings.Contains(strings.Join(os.Args, " "), " list") {
			parseLegacyArgs(args)
			plugin = strings.ToLower(plugin)
			if _, ok := local.PluginMap[plugin]; !ok {
				Failed("Plugin %s is not registered", plugin)
			}
		}
	},
}

func init() {
	LocalDevCmd.PersistentFlags().StringVarP(&plugin, "plugin", "p", "legacy", "Choose which plugin to use")
	RootCmd.AddCommand(LocalDevCmd)
}

func parseLegacyArgs(args []string) {
	activeApp = cfg.ActiveApp
	activeDeploy = cfg.ActiveDeploy
	appClient = cfg.Client

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
}
