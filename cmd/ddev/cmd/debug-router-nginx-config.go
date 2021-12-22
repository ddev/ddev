package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/util"
	"os"
	"strings"

	"github.com/drud/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// DebugRouterNginxConfigCmd implements the ddev debug router-config command
var DebugRouterNginxConfigCmd = &cobra.Command{
	Use:     "router-nginx-config",
	Short:   "Prints the nginx config of the router",
	Example: "ddev debug router-nginx-config",
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
	DebugCmd.AddCommand(DebugRouterNginxConfigCmd)
}
