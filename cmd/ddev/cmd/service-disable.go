package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// ServiceDisable implements the ddev service disable command
var ServiceDisable = &cobra.Command{
	Use:        "disable service [project]",
	Short:      "The service commands have been deprecated and removed and replaced by ddev add-on",
	Hidden:     true,
	Deprecated: `true`,
	Run: func(_ *cobra.Command, args []string) {
		util.Warning("The service commands have been deprecated and removed and replaced by ddev add-on")
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
		err = os.MkdirAll(app.GetConfigPath(disabledServicesDir), 0755)
		if err != nil {
			util.Failed("Unable to create %s: %v", app.GetConfigPath(disabledServicesDir), err)
		}

		if !fileutil.FileExists(app.GetConfigPath(fName)) {
			util.Failed("No file named %s was found in %s", fName, app.GetConfigPath(""))
		}
		err = os.Remove(app.GetConfigPath(disabledServicesDir + "/" + fName))
		if err != nil /*&& err != os.PathError*/ {
			if _, ok := err.(*fs.PathError); !ok {
				util.Failed("Unable to remove %s: %v", app.GetConfigPath(disabledServicesDir+"/"+fName), err)
			}
		}
		err = fileutil.CopyFile(app.GetConfigPath(fName), app.GetConfigPath(disabledServicesDir+"/"+fName))
		if err != nil {
			util.Failed("Unable to disable service %s: %v", serviceName, err)
		}
		err = os.Remove(app.GetConfigPath(fName))
		if err != nil {
			util.Failed("Unable to remove former service file %s: %v", fName, err)
		}

		util.Success("Disabled service %s, use `ddev restart` to see results.", serviceName)
	},
}

func init() {
	ServiceCmd.AddCommand(ServiceDisable)
}
