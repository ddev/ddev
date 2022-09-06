package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"os"
	"path/filepath"
	"regexp"
)

// isCraftCmsApp returns true if the app is of type craftcms
func isCraftCmsApp(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, "craft"))
}

// Set the Docroot to web
func craftCmsConfigOverrideAction(app *DdevApp) error {
	app.Docroot = filepath.Join(app.AppRoot, "web")
	return nil
}

// Set up the .env file for ddev
func craftCmsPostConfigAction(app *DdevApp) error {
	var err error

	var envFilePath string
	envFilePath = filepath.Join(app.AppRoot, ".env")
	// If the .env file doesn't exist, try to create it by copying .env.example to .env
	if !fileutil.FileExists(envFilePath) {
		var exampleEnvFilePath = filepath.Join(app.AppRoot, ".env.example")
		if fileutil.FileExists(exampleEnvFilePath) {
			util.Warning("Copying .env.example to .env")
			err = fileutil.CopyFile(exampleEnvFilePath, envFilePath)
			if err != nil {
				util.Error("Error copying .env.example to .env")
				return err
			}
		} else {
			util.Warning("No .env.example nor .env file exists, you'll need to create your own .env file")
			return nil
		}
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
	siteURLRegEx = regexp.MustCompile(`(PRIMARY_SITE_URL|SITE_URL)=(.*)`)
	siteURLReplace = fmt.Sprintf("$1=%sddev.site", app.GetHTTPSURL())
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

	var composerDefaultFilePath string
	composerDefaultFilePath = filepath.Join(app.AppRoot, "composer.json.default")
	// If composer.json.default exists, rename it to composer.json
	if fileutil.FileExists(composerDefaultFilePath) {
		var composerFilePath string
		composerFilePath = filepath.Join(app.AppRoot, "composer.json")
		util.Warning("Renaming composer.json.default to composer.json")
		err = os.Rename(composerDefaultFilePath, composerFilePath)
		if err != nil {
			util.Error("Error renaming composer.json.default to composer.json")
			return err
		}
	}

	return nil
}

// Currently a placeholder, for possible future expansion
func craftCmsPostStartAction(app *DdevApp) error {
	return nil
}
