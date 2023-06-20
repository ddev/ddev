package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// isCraftCmsApp returns true if the app is of type craftcms
func isCraftCmsApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.ComposerRoot, "craft"))
}

// craftCmsImportFilesAction defines the workflow for importing project files.
func craftCmsImportFilesAction(app *DdevApp, uploadDir, importPath, extPath string) error {
	destPath := app.calculateHostUploadDirFullPath(uploadDir)

	// parent of destination dir should exist
	if !fileutil.FileExists(filepath.Dir(destPath)) {
		return fmt.Errorf("unable to import to %s: parent directory does not exist", destPath)
	}

	// parent of destination dir should be writable.
	if err := os.Chmod(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// If the destination path exists, remove it as was warned
	if fileutil.FileExists(destPath) {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to cleanup %s before import: %v", destPath, err)
		}
	}

	if isTar(importPath) {
		if err := archive.Untar(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	if isZip(importPath) {
		if err := archive.Unzip(importPath, destPath, extPath); err != nil {
			return fmt.Errorf("failed to extract provided archive: %v", err)
		}

		return nil
	}

	//nolint: revive
	if err := fileutil.CopyDir(importPath, destPath); err != nil {
		return err
	}

	return nil
}

// Set up the .env file for ddev
func craftCmsPostStartAction(app *DdevApp) error {
	// If settings management is disabled, do nothing
	if app.DisableSettingsManagement {
		return nil
	}

	// If the .env file doesn't exist, try to create it by copying .env.example to .env
	envFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, ".env")
	if !fileutil.FileExists(envFilePath) {
		var exampleEnvFilePaths = []string{".env.example", ".env.example.dev"}
		for _, envFileName := range exampleEnvFilePaths {
			exampleEnvFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, envFileName)
			if fileutil.FileExists(exampleEnvFilePath) {
				util.Success(fmt.Sprintf("Copied %s to .env", envFileName))
				err := fileutil.CopyFile(exampleEnvFilePath, envFilePath)
				if err != nil {
					util.Error(fmt.Sprintf("Error copying %s to .env", exampleEnvFilePath))

					return err
				}
			}
		}
	}
	// If the .env file *still* doesn't exist, return early
	if !fileutil.FileExists(envFilePath) {
		return nil
	}
	// Read in the .env file
	envMap, envText, err := ReadProjectEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read .env file: %v", err)
	}

	port := "3306"
	driver := "mysql"
	if app.Database.Type == nodeps.Postgres {
		driver = "pgsql"
		port = "5432"
	}

	// If they have older version of .env with DB_DRIVER, DB_SERVER etc, use those
	if _, ok := envMap["DB_SERVER"]; ok {
		envMap = map[string]string{
			"DB_DRIVER":             driver,
			"DB_SERVER":             "db",
			"DB_PORT":               port,
			"DB_DATABASE":           "db",
			"DB_USER":               "db",
			"DB_PASSWORD":           "db",
			"MAILHOG_SMTP_HOSTNAME": "127.0.0.1",
			"MAILHOG_SMTP_PORT":     "1025",
			"PRIMARY_SITE_URL":      app.GetPrimaryURL(),
		}
	} else {
		// Otherwise use the current CRAFT_DB_SERVER etc.
		envMap = map[string]string{
			"CRAFT_DB_DRIVER":       driver,
			"CRAFT_DB_SERVER":       "db",
			"CRAFT_DB_PORT":         port,
			"CRAFT_DB_DATABASE":     "db",
			"CRAFT_DB_USER":         "db",
			"CRAFT_DB_PASSWORD":     "db",
			"MAILHOG_SMTP_HOSTNAME": "127.0.0.1",
			"MAILHOG_SMTP_PORT":     "1025",
			"PRIMARY_SITE_URL":      app.GetPrimaryURL(),
		}
	}

	err = WriteProjectEnvFile(envFilePath, envMap, envText)
	if err != nil {
		return err
	}

	// If composer.json.default exists, rename it to composer.json
	composerDefaultFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, "composer.json.default")
	if fileutil.FileExists(composerDefaultFilePath) {
		composerFilePath := filepath.Join(app.AppRoot, app.ComposerRoot, "composer.json")
		util.Warning("Renaming composer.json.default to composer.json")
		err = os.Rename(composerDefaultFilePath, composerFilePath)
		if err != nil {
			util.Error("Error renaming composer.json.default to composer.json")

			return err
		}
	}

	return nil
}

func craftCmsConfigOverrideAction(app *DdevApp) error {
	app.PHPVersion = nodeps.PHP81
	app.Database = DatabaseDesc{nodeps.MySQL, nodeps.MySQL80}
	return nil
}
