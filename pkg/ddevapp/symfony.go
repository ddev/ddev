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
	return nil
}

// getPHPUploadDir will return a custom upload dir if defined
func getSymfonyUploadDir(app *DdevApp) string {
	return app.UploadDir
}

// symfonyConfigOverrideAction overrides php_version for Symfony 6, requires PHP8.1
func symfonyConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP82
	port := "3306"
    dbConnection := "mysql"
    if app.Database.Type == nodeps.Postgres {
        dbConnection = "pgsql"
        port = "5432"
    }
    app.envMap := map[string]string{
        "DATABASE_URL": dbConnection + "://db:db@db:" + port + "/db",
    }
	return nil
}
