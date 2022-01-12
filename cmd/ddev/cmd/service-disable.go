package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
)

// ServiceDisable implements the ddev service disable command
var ServiceDisable = &cobra.Command{
	Use:     "disable [service] [project]",
	Short:   "disable a 3rd party service",
	Long:    `disable a 3rd party service. The docker-compose.*.yaml will be moved from .ddev into .ddev/services.`,
	Example: `ddev service disable solr`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			util.Failed("You must specify a service to disable")
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
		err = os.MkdirAll(app.GetConfigPath("services"), 0755)
		if err != nil {
			util.Failed("Unable to create %s: %v", app.GetConfigPath("services"), err)
		}

		if !fileutil.FileExists(app.GetConfigPath(fName)) {
			util.Failed("Service %s does not currently exist in .ddev directory", serviceName)
		}
		err = os.Remove(app.GetConfigPath("services/" + fName))
		if err != nil /*&& err != os.PathError*/ {
			if _, ok := err.(*fs.PathError); !ok {
				util.Failed("Unable to remove %s: %v", app.GetConfigPath("services/"+fName), err)
			}
		}
		err = fileutil.CopyFile(app.GetConfigPath(fName), app.GetConfigPath("services/"+fName))
		if err != nil {
			util.Failed("Unable to disable service %s: %v", serviceName, err)
		}
		err = os.Remove(app.GetConfigPath(fName))
		if err != nil {
			util.Failed("Unable to remove former service file %s: %v", fName, err)
		}

		util.Success("disabled service %s, use `ddev restart` to see results.", serviceName)
	},
}

func init() {
	ServiceCmd.AddCommand(ServiceDisable)
}
