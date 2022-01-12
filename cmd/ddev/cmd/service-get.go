package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ServiceGet implements the ddev service get command
var ServiceGet = &cobra.Command{
	Use:     "get servicename [project]",
	Short:   "Get/Download a 3rd party service",
	Long:    `Get/Download a 3rd party service. The service must exist as .ddev/services/docker-compose.<service>.yaml.`,
	Example: `ddev service get rfay/solr`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			util.Failed("You must specify a service to enable")
		}
		apps, err := getRequestedProjects(args[1:], false)
		if err != nil {
			util.Failed("Unable to get project(s) %v: %v", args, err)
		}
		if len(apps) == 0 {
			util.Failed("No project(s) found")
		}
		app := apps[0]
		serviceRepo := args[0]
		fName := fmt.Sprintf("docker-compose.%s.yaml", serviceName)
		if fileutil.FileExists(app.GetConfigPath(fName)) {
			util.Warning("Service %s already enabled, overwriting it, see %s", serviceName, fName)
		}
		err = util.DownloadFile(app.GetConfigPath(fName), serviceRepo, true)
		if err != nil {
			util.Failed("Download of %s failed: %v", serviceRepo, err)
		}
		util.Success("Downloaded and enabled service %s, use `ddev restart` to turn it on.", serviceName)
	},
}

func init() {
	ServiceCmd.AddCommand(ServiceGet)
}
