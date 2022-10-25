package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/archive"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"regexp"
)

// isCraftCmsApp returns true if the app is of type craftcms
func isCraftCmsApp(app *DdevApp) bool {
	if app.ComposerRoot != "" {
		return fileutil.FileExists(filepath.Join(app.ComposerRoot, "craft"))
	}

	return fileutil.FileExists(filepath.Join(app.AppRoot, "craft"))
}

// Set the Docroot to web
func craftCmsConfigOverrideAction(app *DdevApp) error {
	if app.Docroot == "" {
		app.Docroot = "web"
	}

	return nil
}

// Returns the upload directory for importing files, if not already set
func getCraftCmsUploadDir(app *DdevApp) string {
	app.UploadDir = "files"

	return app.UploadDir
}

// craftCmsImportFilesAction defines the workflow for importing project files.
func craftCmsImportFilesAction(app *DdevApp, importPath, extPath string) error {
	if app.UploadDir == "" {
		return errors.Errorf("No upload_dir is set for this (craftcms) project")
	}
	destPath := app.GetHostUploadDirFullPath()

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

// Currently a placeholder, for possible future expansion
func craftCmsPostConfigAction(app *DdevApp) error {
	return nil
}

// Set up the .env file for ddev
func craftCmsPostStartAction(app *DdevApp) error {
	// If settings management is disabled, do nothing
	if app.DisableSettingsManagement {
		return nil
	}
	// If the .env file doesn't exist, try to create it by copying .env.example to .env
	var err error
	var envFileRoot string
	var envFilePath string
	envFileRoot = app.AppRoot
	if app.ComposerRoot != "" {
		envFileRoot = app.ComposerRoot
	}
	envFilePath = filepath.Join(envFileRoot, ".env")
	if !fileutil.FileExists(envFilePath) {
		var exampleEnvFilePaths = []string{".env.example", ".env.example.dev"}
		for _, envFileName := range exampleEnvFilePaths {
			var exampleEnvFilePath = filepath.Join(envFileRoot, envFileName)
			if fileutil.FileExists(exampleEnvFilePath) {
				util.Warning(fmt.Sprintf("Copying %s to .env", envFileName))
				err = fileutil.CopyFile(exampleEnvFilePath, envFilePath)
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
	var envFileContents string
	envFileContents, err = fileutil.ReadFileIntoString(envFilePath)
	if err != nil {
		util.Error("Error reading .env file")

		return err
	}
	// Set the database-related .env variables appropriately for ddev
	var dbRegEx *regexp.Regexp
	dbRegEx = regexp.MustCompile(`DB_(SERVER|DATABASE|USER|PASSWORD)=(.*)`)
	envFileContents = dbRegEx.ReplaceAllString(envFileContents, `DB_$1=db`)
	// Set the primary site URL
	var siteURLRegEx *regexp.Regexp
	var siteURLReplace string
	siteURLRegEx = regexp.MustCompile(`PRIMARY_SITE_URL=(.*)`)
	siteURLReplace = fmt.Sprintf("PRIMARY_SITE_URL=%s", app.GetHTTPSURL())
	if !siteURLRegEx.MatchString(envFileContents) {
		envFileContents += "\nPRIMARY_SITE_URL="
	}
	envFileContents = siteURLRegEx.ReplaceAllString(envFileContents, siteURLReplace)
	// Set the MailHog .env variables (https://ddev.readthedocs.io/en/latest/users/basics/developer-tools/#email-capture-and-review-mailhog)
	var mailhogRegEx *regexp.Regexp
	mailhogRegEx = regexp.MustCompile(`(MAILHOG_SMTP_HOSTNAME|MAILHOG_SMTP_PORT)=(.*)`)
	if !mailhogRegEx.MatchString(envFileContents) {
		envFileContents += "\n\nMAILHOG_SMTP_HOSTNAME=localhost\nMAILHOG_SMTP_PORT=1025"
	}
	// Write the modified .env file out
	var f *os.File
	f, err = os.Create(".env")
	if err != nil {
		util.Error("Error creating .env file")

		return err
	}
	_, err = f.WriteString(envFileContents)
	if err != nil {
		util.Error("Error writing .env file")

		return err
	}
	// If composer.json.default exists, rename it to composer.json
	var composerFileRoot string
	var composerDefaultFilePath string
	composerFileRoot = app.AppRoot
	if app.ComposerRoot != "" {
		composerFileRoot = app.ComposerRoot
	}
	composerDefaultFilePath = filepath.Join(composerFileRoot, "composer.json.default")
	if fileutil.FileExists(composerDefaultFilePath) {
		var composerFilePath string
		composerFilePath = filepath.Join(composerFileRoot, "composer.json")
		util.Warning("Renaming composer.json.default to composer.json")
		err = os.Rename(composerDefaultFilePath, composerFilePath)
		if err != nil {
			util.Error("Error renaming composer.json.default to composer.json")

			return err
		}
	}

	return nil
}
