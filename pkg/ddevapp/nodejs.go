package ddevapp

import "github.com/ddev/ddev/pkg/nodeps"

func nodejsConfigOverrideAction(app *DdevApp) error {
	app.WebserverType = nodeps.WebserverNginxNodeJS
	return nil
}
