package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
)

func getRequestedApps(args []string, allFlag bool) ([]*ddevapp.DdevApp, error) {
	// If allFlag is true, all active apps will be returned.
	if allFlag {
		if len(args) > 0 {
			return []*ddevapp.DdevApp{}, fmt.Errorf("too many arguments provided with the --all flag")
		}

		return ddevapp.GetApps(), nil
	}

	// If multiple apps are requested by their site name, collect and return them.
	if len(args) > 0 {
		var apps []*ddevapp.DdevApp

		for _, siteName := range args {
			app, err := ddevapp.GetActiveApp(siteName)
			if err != nil {
				util.Warning("Failed to get %s: %v", siteName, err)
			} else {
				apps = append(apps, app)
			}
		}

		return apps, nil
	}

	// If the allFlag is false and no specific apps are requested, return the current app.
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return []*ddevapp.DdevApp{}, err
	}

	return []*ddevapp.DdevApp{app}, nil
}
