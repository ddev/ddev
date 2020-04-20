package ddevapp

import (
	"fmt"
)

func phpPostStartAction(app *DdevApp) error {
	if !app.DisableSettingsManagement {
		if _, err := app.CreateSettingsFile(); err != nil {
			return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
		}
	}
	return nil
}
