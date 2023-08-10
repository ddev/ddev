package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
)

// see https://github.com/ddev/ddev/pull/5058 if the bin folder changes.
// isSilverstripeApp returns true if the app is of type Silverstripe
func isSilverstripeApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.ComposerRoot, "vendor/bin/sake"))
}

func silverstripePostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read .env file: %v", err)
	}
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, app.ComposerRoot, ".env.example"), envFilePath)
		if err != nil {
			util.Debug("Silverstripe: .env.example does not exist yet, not trying to process it")
			return nil
		}
		_, envText, err = ReadProjectEnvFile(envFilePath)
		if err != nil {
			return err
		}
	}
	port := "3306"
	dbConnection := "MySQLDatabase"
	// Although possible, it is extremely uncommon to use Postgres with Silverstripe.
	// Thus the option is there to override
	if app.Database.Type == nodeps.Postgres {
		dbConnection = "PostgreSQLDatabase"
		port = "5432"
	}
	envMap := map[string]string{
		"SS_DATABASE_SERVER":        "db",
		"SS_DATABASE_PORT":          port,
		"SS_DATABASE_NAME":          "db",
		"SS_DATABASE_USERNAME":      "db",
		"SS_DATABASE_PASSWORD":      "db",
		"SS_ENVIRONMENT_TYPE":       "dev",
		"SS_DATABASE_CLASS":         dbConnection,
		"MAILER_DSN":                "smtp://localhost:1025",
		"SS_DEFAULT_ADMIN_USERNAME": "admin",
		"SS_DEFAULT_ADMIN_PASSWORD": "password",
	}
	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}

// silverstripeConfigOverrideAction overrides php_version for Silverstripe, requires PHP8.1
func silverstripeConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP81
	app.WebserverType = nodeps.WebserverApacheFPM
	app.Database.Type = nodeps.MariaDB
	app.Database.Version = nodeps.MariaDB104
	return nil
}
