package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

func symfonyPostStartAction(app *DdevApp) error {
	if !app.DisableSettingsManagement {
		if _, err := app.CreateSettingsFile(); err != nil {
			return fmt.Errorf("failed to write settings file %s: %v", app.SiteDdevSettingsFile, err)
		}
	}
	return nil
}

func symfonyPostStartAction(app *DdevApp) error {
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
		"DATABASE_URL": fmt.Printf("%c://db:db@db:%p/db", dbConnection, port)
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




