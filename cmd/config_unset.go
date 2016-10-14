package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// unsetCmd
var unsetCmd = &cobra.Command{
	Use:   "unset config_item [, config_item...]",
	Short: "Set configuration values for DRUD.",
	Long:  `Set configuration values for DRUD.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("You must pick a config item to unset.")
		}

		for _, arg := range args {

			if arg == "protocol" {
				cfg.Protocol = ""
			} else if arg == "drud_host" {
				cfg.DrudHost = ""
			} else if arg == "version" {
				cfg.Version = ""
			} else if arg == "client" {
				cfg.Client = ""
			} else if arg == "active_app" {
				cfg.ActiveApp = ""
			} else if arg == "active_deploy" {
				cfg.ActiveDeploy = ""
			} else if arg == "disable_updates" {
				cfg.Dev = false
			} else {
				log.Fatalf("There is no config item called %s.", arg)
			}
		}

		err := cfg.WriteConfig(drudconf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Config items unset.")
	},
}

func init() {
	ConfigCmd.AddCommand(unsetCmd)
}
