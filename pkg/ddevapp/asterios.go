package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// isAsteriosApp returns true if the app is of type asterios
func isAsteriosApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.ComposerRoot, "asterios"))
}

// asteriosPostStartAction checks to see if the .env file is set up
func asteriosPostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to read .env file: %v", err)
	}
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, app.ComposerRoot, ".env.example"), filepath.Join(app.AppRoot, app.ComposerRoot, ".env"))
		if err != nil {
			util.Debug("Asterios: .env.example does not exist yet, not trying to process it")
			return nil
		}
		_, envText, err = ReadProjectEnvFile(envFilePath)
		if err != nil {
			return err
		}
	}
	envMap := map[string]string{
		"APP_URL":     app.GetPrimaryURL(),
		"MAIL_MAILER": "smtp",
		"MAIL_HOST":   "127.0.0.1",
		"MAIL_PORT":   "1025",
	}

	// Only set database configuration if db container is not omitted
	shouldManageDB := !slices.Contains(app.OmitContainers, "db")

	// PostgreSQL is not supported at the time
	if app.Database.Type == nodeps.Postgres {
		shouldManageDB = false
	}

	if shouldManageDB {
		port := "3306"

		envMap["DB_HOST"] = "db"
		envMap["DB_PORT"] = port
		envMap["DB_DATABASE"] = "db"
		envMap["DB_USERNAME"] = "db"
		envMap["DB_PASSWORD"] = "db"
	}
	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}
