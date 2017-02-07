package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// unsetCmd
var (
	uAPIVersion      bool
	uActiveApp       bool
	uActiveDeploy    bool
	uClient          bool
	uDrudHost        bool
	uGithubAuthToken bool
	uGithubAuthOrg   bool
	uProtocol        bool
	uVaultAddr       bool
	uVaultAuthToken  bool
	unset            bool
)
var unsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Unset configuration values for DRUD.",
	Long:  `Unset configuration values for DRUD.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(os.Args) < 4 {
			cmd.Usage()
			os.Exit(0)
		}
		if !isFlagPresent(cmd) {
			cmd.Usage()
			log.Fatalln("No configuration flag provided. Use one of the available flags to unset a configuration value.")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if uAPIVersion {
			cfg.APIVersion = ""
			unset = true
		}
		if uActiveApp {
			cfg.ActiveApp = ""
			unset = true
		}
		if uActiveDeploy {
			cfg.ActiveDeploy = ""
			unset = true
		}
		if uClient {
			cfg.Client = ""
			unset = true
		}
		if uDrudHost {
			cfg.DrudHost = ""
			unset = true
		}
		if uGithubAuthToken {
			cfg.GithubAuthToken = ""
			unset = true
		}
		if uGithubAuthOrg {
			cfg.GithubAuthOrg = ""
			unset = true
		}
		if uProtocol {
			cfg.Protocol = ""
			unset = true
		}
		if uVaultAddr {
			cfg.VaultAddr = ""
			unset = true
		}
		if uVaultAuthToken {
			cfg.VaultAuthToken = ""
			unset = true
		}

		err := cfg.WriteConfig(cfgFile)
		if err != nil {
			log.Fatal(err)
		}

		if unset == true {
			fmt.Println("Config items unset.")
		} else {
			fmt.Println("Config items not unset. See `drud config unset --help`")
		}
	},
}

func init() {
	unsetCmd.Flags().BoolVarP(&uAPIVersion, "apiversion", "", false, "Unset APIVersion")
	unsetCmd.Flags().BoolVarP(&uActiveApp, "activeapp", "", false, "Unset ActiveApp")
	unsetCmd.Flags().BoolVarP(&uActiveDeploy, "activedeploy", "", false, "Unset ActiveDeploy")
	unsetCmd.Flags().BoolVarP(&uClient, "client", "", false, "Unset Client")
	unsetCmd.Flags().BoolVarP(&uDrudHost, "drudhost", "", false, "Unset DrudHost")
	unsetCmd.Flags().BoolVarP(&uGithubAuthToken, "githubauthtoken", "", false, "Unset GithubAuthToken")
	unsetCmd.Flags().BoolVarP(&uGithubAuthOrg, "githubauthorg", "", false, "Unset GithubAuthOrg")
	unsetCmd.Flags().BoolVarP(&uProtocol, "protocol", "", false, "Unset Protocol")
	unsetCmd.Flags().BoolVarP(&uVaultAddr, "vaultaddr", "", false, "Unset VaultAddr")
	unsetCmd.Flags().BoolVarP(&uVaultAuthToken, "vaultauthtoken", "", false, "Unset VaultAuthToken")

	ConfigCmd.AddCommand(unsetCmd)
}
