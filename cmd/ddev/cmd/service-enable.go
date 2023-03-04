package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

const disabledServicesDir = ".disabled-services"

// ServiceEnable implements the ddev service enable command
var ServiceEnable = &cobra.Command{
	Use:     "enable service [project]",
	Short:   "Enable a 3rd party service",
	Long:    fmt.Sprintf(`Enable a 3rd party service. The service must exist as .ddev/%s/docker-compose.<service>.yaml. Note that you can use "ddev get" to obtain a service not already on your system.`, disabledServicesDir),
	Example: `ddev service enable solr`,
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
		serviceName := args[0]
		fName := fmt.Sprintf("docker-compose.%s.yaml", serviceName)
		if fileutil.FileExists(app.GetConfigPath(fName)) {
			util.Failed("Service %s already enabled, see %s", serviceName, fName)
		}
		if !fileutil.FileExists(app.GetConfigPath(disabledServicesDir + "/" + fName)) {
			util.Failed("No %s found in %s", fName, app.GetConfigPath(disabledServicesDir))
		}
		err = fileutil.CopyFile(app.GetConfigPath(disabledServicesDir+"/"+fName), app.GetConfigPath(fName))
		if err != nil {
			util.Failed("Unable to enable service %s: %v", serviceName, err)
		}
		util.Success("Enabled service %s, use `ddev restart` to turn it on.", serviceName)
	},
}

func init() {
	ServiceCmd.AddCommand(ServiceEnable)
}
