package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

var (
	activeApp       string
	activeDeploy    string
	apiVersion      string
	client          string
	drudHost        string
	githubAuthToken string
	githubAuthOrg   string
	protocol        string
	vaultAddr       string
	workspace       string
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values for DRUD.",
	Long:  `Set configuration values for DRUD.`,
	Run: func(cmd *cobra.Command, args []string) {
		if activeApp != "" {
			cfg.ActiveApp = activeApp
		}

		if activeDeploy != "" {
			cfg.ActiveDeploy = activeDeploy
		}

		if apiVersion != "" {
			cfg.APIVersion = apiVersion
		}

		if client != "" {
			cfg.Client = client
		}

		if drudHost != "" {
			cfg.DrudHost = drudHost
		}

		if githubAuthToken != "" {
			cfg.GithubAuthToken = githubAuthToken
		}

		if githubAuthOrg != "" {
			cfg.GithubAuthOrg = githubAuthOrg
		}

		if protocol != "" {
			cfg.Protocol = protocol
		}

		if vaultAddr != "" {
			cfg.VaultAddr = vaultAddr
		}

		if workspace != "" {
			if strings.HasPrefix(workspace, "$HOME") || strings.HasPrefix(workspace, "~") {
				workspace = strings.Replace(workspace, "$HOME", homedir, 1)
				workspace = strings.Replace(workspace, "~", homedir, 1)
			}
			cfg.Workspace = workspace
		}

		err := cfg.WriteConfig(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Config items set.")
	},
}

func init() {
	setCmd.Flags().StringVarP(&activeApp, "activeapp", "", "", "active app name")
	setCmd.Flags().StringVarP(&activeDeploy, "activedeploy", "", "", "active deploy name")
	setCmd.Flags().StringVarP(&apiVersion, "apiversion", "", "v0.1", "API Version")
	setCmd.Flags().StringVarP(&client, "client", "", "", "DRUD client name")
	setCmd.Flags().StringVarP(&drudHost, "drudhost", "", "", "DRUD API hostname. e.g. drudapi.example.com")
	setCmd.Flags().StringVarP(&githubAuthToken, "githubauthtoken", "", "", "GitHub Authorization Token")
	setCmd.Flags().StringVarP(&githubAuthOrg, "githubauthorg", "", "", "GitHub Authorization Organization")
	setCmd.Flags().StringVarP(&protocol, "protocol", "", "https", "Protocol to use, e.g. http or https")
	setCmd.Flags().StringVarP(&vaultAddr, "vaultaddr", "", "https://vault.drud.com:8200", "Vault Host")
	setCmd.Flags().StringVarP(&workspace, "workspace", "", "$HOME/.drud", "Local workspace for drud dev.")

	ConfigCmd.AddCommand(setCmd)
}
