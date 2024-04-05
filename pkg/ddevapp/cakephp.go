package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
	"strings"
)

// isCakephpApp returns true if the app is of type cakephp
func isCakephpApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "bin/cake.php"))
}

func cakephpPostStartAction(app *DdevApp) error {
	// We won't touch env if disable_settings_management: true
	if app.DisableSettingsManagement {
		return nil
	}
	envFileName := "config/.env"
	envFilePath := filepath.Join(app.AppRoot, envFileName)
	_, _, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to read .env file in config folder: %v", err)
	}
	if err == nil {
		envFileName = "config/.env.ddev"
		envFilePath = filepath.Join(app.AppRoot, envFileName)
		_, _, err = ReadProjectEnvFile(envFilePath)
		if err == nil {
			util.Warning("CakePHP: .env.ddev file exists already. Replacing it. You can rename it or copy settings to your .env file.")
		} else {
			util.Warning("CakePHP: .env file exists already. Creating .env.ddev. You can rename it or copy settings to your .env file.")
		}
	} else {
		util.Success("CakePHP: Creating .env file to store your config settings.")
	}
	err = fileutil.CopyFile(filepath.Join(app.AppRoot, "config/.env.example"), envFilePath)
	if err != nil {
		util.Debug("CakePHP: .env.example does not exist yet in config folder, not trying to process it")
		return nil
	}
	_, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil {
		return err
	}
	port := "3306"
	dbConnection := "mysql"
	if app.Database.Type == nodeps.Postgres {
		dbConnection = "pgsql"
		port = "5432"
	}
	envMap := map[string]string{
		"export APP_NAME":                    app.GetName(),
		"export DEBUG":                       "true",
		"export APP_ENCODING":                "UTF-8",
		"export APP_DEFAULT_LOCALE":          "en_US",
		"export DATABASE_URL":                dbConnection + "://db:db@db:" + port + "/db",
		"export EMAIL_TRANSPORT_DEFAULT_URL": "smtp://localhost:1025",
		"export SECURITY_SALT":               util.HashSalt(app.GetName()),
		"export DEBUG_KIT_SAFE_TLD":          "site",
	}
	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}
	err = enableDotEnvLoading(app)
	if err != nil {
		return err
	}
	return nil
}

// cakephpConfigOverrideAction overrides php_version for CakePHP, requires PHP8.1
func cakephpConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP83
	app.DisableUploadDirsWarning = true
	return nil
}

func enableDotEnvLoading(app *DdevApp) error {
	bootstrapFileName := "config/bootstrap.php"
	bootstrapFilePath := filepath.Join(app.AppRoot, bootstrapFileName)
	envFunction := "// if (!env('APP_NAME') && file_exists(CONFIG . '.env')) {\n//     $dotenv = new \\josegonzalez\\Dotenv\\Loader([CONFIG . '.env']);\n//     $dotenv->parse()\n//         ->putenv()\n//         ->toEnv()\n//         ->toServer();\n// }\n"
	err := fileutil.ReplaceStringInFile(envFunction, strings.ReplaceAll(envFunction, "// ", ""), bootstrapFilePath, bootstrapFilePath)
	if err != nil {
		return err
	}

	return nil
}
