package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// DebugRouterNginxConfigCmd implements the ddev debug router-config command
// This is only for the obsolete traditional router
var DebugRouterNginxConfigCmd = &cobra.Command{
	Use:     "traditional-router-nginx-config",
	Short:   "Obsolete: Prints the nginx config in the traditional router",
	Example: "ddev debug traditional-router-nginx-config",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to debug router-config : %v", err)
		}

		if ddevapp.IsRouterDisabled(app) {
			util.Warning("Router is disabled by config")
			os.Exit(0)
		}

		container, _ := ddevapp.FindDdevRouter()

		if container == nil {
			util.Failed("Router is not running")
			return // extraneous - just for lint to know we exited
		}

		// see containers/ddev-router/testtools/testgen.sh
		stdout, _, err := dockerutil.Exec(container.ID, "cat /etc/nginx/conf.d/ddev.conf", "")

		if err != nil {
			util.Failed("Failed to run docker-gen command in ddev-router container: %v", err)
		}

		output.UserOut.Print(strings.TrimSpace(stdout))
	},
}

func init() {
	if globalconfig.DdevGlobalConfig.Router != nodeps.RouterTypeTraefik {
		DebugCmd.AddCommand(DebugRouterNginxConfigCmd)
	}
}
