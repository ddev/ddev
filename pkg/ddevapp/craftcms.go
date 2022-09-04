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
	var envFilePath string
	var err error
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
	} else {
		util.Warning(".env file present")
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
	// Set the default site URL
	var siteUrlRegEx *regexp.Regexp
	var siteUrlReplace string
	siteUrlRegEx = regexp.MustCompile(`(DEFAULT_SITE_URL|SITE_URL)=(.*)`)
	siteUrlReplace = fmt.Sprintf("$1=%sddev.site", app.GetHTTPSURL())
	if !siteUrlRegEx.MatchString(envFileContents) {
		envFileContents += "\nDEFAULT_SITE_URL="
	}
	envFileContents = siteUrlRegEx.ReplaceAllString(envFileContents, siteUrlReplace)
	// Set the MailHog .env variables (https://ddev.readthedocs.io/en/latest/users/basics/developer-tools/#email-capture-and-review-mailhog)
	var mailhogRegEx *regexp.Regexp
	mailhogRegEx = regexp.MustCompile(`(MAILHOG_SMTP_HOSTNAME|MAILHOG_SMTP_PORT)=(.*)`)
	if !mailhogRegEx.MatchString(envFileContents) {
	    envFileContents += "\nMAILHOG_SMTP_HOSTNAME=localhost\nMAILHOG_SMTP_PORT=1025"
	}
	// Write the modified .env file out
	var f *os.File
	f, err = os.Create(".env")
	_, err = f.WriteString(envFileContents)
	if err != nil {
		util.Error("Error writing .env file")
		return err
	}

	return nil
}

// Currently a placeholder, for possible future expansion
func craftCmsPostStartAction(app *DdevApp) error {
	return nil
}
