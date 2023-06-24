package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// TODO: This debug function should be removed by 2024-01-01 since the nginx-proxy router is deprecated
// and since it's easy enough to do this task manually.

// DebugRouterNginxConfigCmd implements the ddev debug router-config command
// This is only for the obsolete nginx-proxy router
var DebugRouterNginxConfigCmd = &cobra.Command{
	Use:     "nginx-proxy-router-nginx-config",
	Short:   "Obsolete: Prints the nginx config in the legacy `nginx-proxy` router",
	Example: "ddev debug nginx-proxy-router-nginx-config",
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
	if globalconfig.DdevGlobalConfig.Router == types.RouterTypeNginxProxy {
		DebugCmd.AddCommand(DebugRouterNginxConfigCmd)
	}
}
