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

const (
	// WarnTypeAbsent warns if type is absent
	WarnTypeAbsent = iota
	// WarnTypeNotConfigured warns if type not configured
	WarnTypeNotConfigured = iota
)

// isLaravelApp returns true if the app is of type laravel
func isLaravelApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "artisan"))
}

func laravelPostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	envFilePath := filepath.Join(app.AppRoot, ".env")
	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to read .env file: %v", err)
	}
	if os.IsNotExist(err) {
		err = fileutil.CopyFile(filepath.Join(app.AppRoot, ".env.example"), filepath.Join(app.AppRoot, ".env"))
		if err != nil {
			util.Debug("Laravel: .env.example does not exist yet, not trying to process it")
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

	if shouldManageDB {
		port := "3306"
		dbConnection := "mariadb"

		if app.Database.Type == nodeps.MariaDB {
			hasMariaDBDriver, _ := fileutil.FgrepStringInFile(filepath.Join(app.AppRoot, "config/database.php"), "mariadb")
			if !hasMariaDBDriver {
				// Older versions of Laravel (before 11) use "mysql" driver for MariaDB
				// This change is required to prevent this error on "php artisan migrate":
				// InvalidArgumentException Database connection [mariadb] not configured
				dbConnection = "mysql"
			}
		} else if app.Database.Type == nodeps.MySQL {
			dbConnection = "mysql"
		} else if app.Database.Type == nodeps.Postgres {
			dbConnection = "pgsql"
			port = "5432"
		}

		envMap["DB_HOST"] = "db"
		envMap["DB_PORT"] = port
		envMap["DB_DATABASE"] = "db"
		envMap["DB_USERNAME"] = "db"
		envMap["DB_PASSWORD"] = "db"
		envMap["DB_CONNECTION"] = dbConnection
	}
	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}
