package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
)

// getRequestedApps will collect and return the requested apps from command line arguments and flags.
func getRequestedApps(args []string, all bool) ([]*ddevapp.DdevApp, error) {
	// If all is true, all active apps will be returned.
	if all {
		return ddevapp.GetApps(), nil
	}

	// If multiple apps are requested by their site name, collect and return them.
	if len(args) > 0 {
		var apps []*ddevapp.DdevApp

		for _, siteName := range args {
			app, err := ddevapp.GetActiveApp(siteName)
			if err != nil {
				return []*ddevapp.DdevApp{}, fmt.Errorf("failed to get %s: %v", siteName, err)
			}

			apps = append(apps, app)
		}

		return apps, nil
	}

	// If all is false and no specific apps are requested, return the current app.
	app, err := ddevapp.GetActiveApp("")
	if err != nil {
		return []*ddevapp.DdevApp{}, err
	}

	return []*ddevapp.DdevApp{app}, nil
}
