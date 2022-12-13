package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"os"
)

func symfonyPostStartAction(app *DdevApp) error {
	if !app.DisableSettingsManagement {
		if _, err := app.CreateSettingsFile(); err != nil {
			return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
		}
	}
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	_, envText, err := ReadProjectEnvFile(app)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read .env file: %v", err)
	}
	port := "3306"
	dbConnection := "mysql"
	if app.Database.Type == nodeps.Postgres {
		dbConnection = "pgsql"
		port = "5432"
	}
	envMap := map[string]string{
		"DATABASE_URL": dbConnection + "://db:db@db:" + port + "/db",
	}
	err = WriteProjectEnvFile(app, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}

// getPHPUploadDir will return a custom upload dir if defined
func getSymfonyUploadDir(app *DdevApp) string {
	return app.UploadDir
}
