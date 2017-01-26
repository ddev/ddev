package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// setCmd represents the set command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show configuration for DRUD.",
	Long:  `Show configuration for DRUD.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cfgFile)
		fmt.Printf("\tapiversion = %+v\n", cfg.APIVersion)
		fmt.Printf("\tactiveapp = %+v\n", cfg.ActiveApp)
		fmt.Printf("\tactivedeploy = %+v\n", cfg.ActiveDeploy)
		fmt.Printf("\tclient = %+v\n", cfg.Client)
		fmt.Printf("\tdrudhost = %+v\n", cfg.DrudHost)
		fmt.Printf("\tgithubauthtoken = %+v\n", cfg.GithubAuthToken)
		fmt.Printf("\tgithubauthorg = %+v\n", cfg.GithubAuthOrg)
		fmt.Printf("\tprotocol = %+v\n", cfg.Protocol)
		fmt.Printf("\tvaultaddr = %+v\n", cfg.VaultAddr)
		fmt.Printf("\tvaultauthtoken = %+v\n", cfg.VaultAuthToken)
		fmt.Printf("\tworkspace = %+v\n", cfg.Workspace)
	},
}

func init() {
	ConfigCmd.AddCommand(showCmd)
}
