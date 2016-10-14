package cmd

import (
	"fmt"
	"log"
	"path"

	"github.com/spf13/cobra"
)

var client, token, repoDir, adminToken, protocol, drudHost, version, activeApp, activeDeploy string
var developerMode bool

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values for DRUD.",
	Long:  `Set configuration values for DRUD.`,
	Run: func(cmd *cobra.Command, args []string) {

		if repoDir == "" {
			repoDir = path.Join(homedir, "Desktop")
		}

		if protocol != "" {
			cfg.Protocol = protocol
		}

		if drudHost != "" {
			cfg.DrudHost = drudHost
		}

		if version != "" {
			cfg.Version = version
		}

		if client != "" {
			cfg.Client = client
		}

		if activeApp != "" {
			cfg.ActiveApp = activeApp
		}

		if activeDeploy != "" {
			cfg.ActiveDeploy = activeDeploy
		}

		cfg.Dev = developerMode

		err := WriteConfig(cfg, drudconf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Config items set.")
	},
}

func init() {
	setCmd.Flags().StringVarP(&protocol, "protocol", "p", "", "Protocol to use, e.g. http or https")
	setCmd.Flags().StringVarP(&drudHost, "drud_host", "o", "", "DRUD API hostname. e.g. drudapi.mydrud.drud.us")
	setCmd.Flags().StringVarP(&version, "version", "v", "", "API Version")
	setCmd.Flags().StringVarP(&client, "client", "c", "", "DRUD client name")
	setCmd.Flags().StringVarP(&activeApp, "active_app", "a", "", "active app name")
	setCmd.Flags().StringVarP(&activeDeploy, "active_deploy", "d", "", "active deploy name")
	setCmd.Flags().BoolVarP(&developerMode, "disable_updates", "u", false, "This will disable auto updates and is not recommended.")
	ConfigCmd.AddCommand(setCmd)
}
