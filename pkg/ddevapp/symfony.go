package ddevapp

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
)

// isSymfonyApp returns true if the app is of type laravel
func isSymfonyApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "bin", "console")) && fileutil.FileExists(filepath.Join(app.AppRoot, "src", "Kernel.php"))
}

func symfonyEnvMailer(app *DdevApp, envMap map[string]string) {
	envMap["MAILER_AUTH_MODE"] = ""
	envMap["MAILER_PASSWORD"] = ""
	envMap["MAILER_USERNAME"] = ""
	envMap["MAILER_CATCHER"] = "1"
	envMap["MAILER_DRIVER"] = "smtp"
	envMap["MAILER_DSN"] = "smtp://localhost:1025"
	envMap["MAILER_HOST"] = "localhost"
	envMap["MAILER_PORT"] = "1025"
	envMap["MAILER_URL"] = "smtp://localhost:1025"
	envMap["MAILER_WEB_URL"] = fmt.Sprintf("%s:8026", app.GetPrimaryURL())
}

func symfonyEnvDatabase(app *DdevApp, envMap map[string]string) {
	if slices.Contains(app.OmitContainers, "db") {
		return
	}

	dbPort := ""
	dbDriver := ""
	dbVersion := ""

	if app.Database.Type == nodeps.Postgres {
		dbPort = "5432"
		dbDriver = "postgres"
		dbVersion = app.Database.Version
	} else if app.Database.Type == nodeps.MariaDB || app.Database.Type == nodeps.MySQL {
		dbPort = "3306"
		dbDriver = "mysql"
		if app.Database.Type == nodeps.MariaDB {
			// doctrine requires mariadb version until its patch version so add 0 as default patch version
			dbVersion = fmt.Sprintf("%s.0-mariadb", app.Database.Version)
		} else {
			dbVersion = app.Database.Version
		}
	}

	if dbVersion != "" {
		envMap["DATABASE_DRIVER"] = dbDriver
		envMap["DATABASE_HOST"] = "db"
		envMap["DATABASE_NAME"] = "db"
		envMap["DATABASE_PASSWORD"] = "db"
		envMap["DATABASE_USER"] = "db"
		envMap["DATABASE_PORT"] = dbPort
		envMap["DATABASE_SERVER"] = fmt.Sprintf("%s://db:%s", dbDriver, dbPort)
		envMap["DATABASE_URL"] = fmt.Sprintf("%s://db:db@db:%s/db?sslmode=disable&charset=utf8&serverVersion=%s", dbDriver, dbPort, dbVersion)
		envMap["DATABASE_VERSION"] = dbVersion
	}
}

// symfonyPostStartAction creates the .env.local file
func symfonyPostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}

	envFilePath := filepath.Join(app.AppRoot, ".env.local")
	// won't throw any error here, as it will be created anyway
	_, envText, _ := ReadProjectEnvFile(envFilePath)

	envMap := make(map[string]string)

	symfonyEnvMailer(app, envMap)
	symfonyEnvDatabase(app, envMap)

	for _, addOn := range GetInstalledAddons(app) {
		if addOn.Name == "redis" {
			envMap["REDIS_HOST"] = "redis"
			envMap["REDIS_PORT"] = "6379"
			envMap["REDIS_SCHEME"] = "redis"
			if addOn.Repository == "ddev/ddev-redis-7" {
				envMap["REDIS_USER"] = "redis"
				envMap["REDIS_PASSWORD"] = "redis"
				envMap["REDIS_URL"] = "redis://redis:redis@redis:6379"
			} else {
				envMap["REDIS_URL"] = "redis://redis:6379"
			}
		}
	}

	err := WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	return nil
}

// getSymfonyHooks for appending as byte array.
func getSymfonyHooks() []byte {
	backdropHooks := `## Un-comment to consume async message.
#    post-start:
#    - exec: symfony run --daemon --watch=config,src,templates,vendor bin/console messenger:consume async -vv
`
	return []byte(backdropHooks)
}
