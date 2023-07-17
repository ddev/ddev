package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const siteSettingsDdevPy = "settings.ddev.py"

// isDjango4App returns true if the app is of type django4
func isDjango4App(app *DdevApp) bool {
	return fileutil.FileExists(filepath.Join(app.AppRoot, app.Docroot, "manage.py"))
}

// django4ConfigOverrideAction sets up webserverType and anything else
// we might need for django.
func django4ConfigOverrideAction(app *DdevApp) error {
	if app.WebserverType == nodeps.WebserverDefault {
		app.WebserverType = nodeps.WebserverNginxGunicorn
	}
	if app.Database == DatabaseDefault {
		app.Database.Type = nodeps.Postgres
		app.Database.Version = nodeps.Postgres14
	}

	return nil
}

// django4PostConfigAction just reminds people that they may need DJANGO_SETTINGS_MODULE env var
func django4PostConfigAction(_ *DdevApp) error {
	util.Warning("Your project may need a DJANGO_SETTINGS_MODULE environment variable to work correctly")
	return nil
}

// django4SettingsCreator creates the settings.ddev.py
func django4SettingsCreator(app *DdevApp) (string, error) {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return "", nil
	}

	err := writeDjango4SettingsDdevPy(app.GetConfigPath("settings/settings.ddev.py"), app)
	if err != nil {
		return "", err
	}
	return app.GetConfigPath("settings/settings.ddev.py"), nil
}

// django4PostStartAction handles creating settings for the project
func django4PostStartAction(app *DdevApp) error {
	// Return early because we aren't expected to manage settings.
	if app.DisableSettingsManagement {
		return nil
	}

	settingsFile, _, err := app.Exec(&ExecOpts{
		Cmd: "find-django-settings-file.py",
	})
	if err != nil {
		util.Warning("Unable to find django settings file: %v", err)
	}

	settingsFile = strings.Trim(settingsFile, "\n")
	app.SiteSettingsPath = strings.TrimPrefix(settingsFile, "/var/www/html/")
	app.SiteSettingsPath = filepath.Join(app.AppRoot, app.SiteSettingsPath)

	app.SiteDdevSettingsFile = "/mnt/ddev_config/settings/settings.ddev.py"

	django4SettingsIncludeStanza := fmt.Sprintf(`

# #ddev-generated code to import DDEV settings
import os
if os.environ.get('IS_DDEV_PROJECT') == 'true':
    from pathlib import Path
    import importlib.util
    import sys

    s = Path('%s')
    if s.is_file():
        spec = importlib.util.spec_from_file_location("ddev_settings", s)
        ddev_settings = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(ddev_settings)

        # Get the current module to set attributes
        current_module = sys.modules[__name__]

        # Add or update attributes from the ddev_settings module
        for name, value in vars(ddev_settings).items():
            if not name.startswith("__"):  # Exclude special attributes
                setattr(current_module, name, value)
# End DDEV-generated code
`, app.SiteDdevSettingsFile)

	included, err := fileutil.FgrepStringInFile(app.SiteSettingsPath, nodeps.DdevFileSignature)
	if err != nil {
		return err
	}

	if included {
		output.UserOut.Printf("Existing %s includes %s", app.SiteSettingsPath, app.SiteDdevSettingsFile)
	} else {
		output.UserOut.Printf("Existing %s file does not include %s, modifying to include ddev settings", app.SiteSettingsPath, app.SiteDdevSettingsFile)

		// Add the inclusion
		file, err := os.OpenFile(app.SiteSettingsPath, os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer util.CheckClose(file)

		_, err = file.Write([]byte(django4SettingsIncludeStanza))
		if err != nil {
			return err
		}

		err = app.MutagenSyncFlush()
		if err != nil {
			return err
		}
	}

	return err
}

// reloadGunicorn hits gunicorn process with HUP, reloading
func reloadGunicorn(app *DdevApp) {
	stdout, stderr, err := app.Exec(&ExecOpts{
		Cmd: "pkill -HUP gunicorn",
	})
	if err != nil {
		util.Warning("Failed to reload gunicorn configuration: stdout='%s', stderr='%s': %v", stdout, stderr, err)
	}
}

// writeDjango4SettingsDdevPy dynamically produces valid settings.ddev.py file by combining a configuration
// object with a data-driven template.
func writeDjango4SettingsDdevPy(filePath string, app *DdevApp) error {
	if fileutil.FileExists(filePath) {
		// Check if the file is managed by ddev.
		signatureFound, err := fileutil.FgrepStringInFile(filePath, nodeps.DdevFileSignature)
		if err != nil {
			return err
		}

		// If the signature wasn't found, warn the user and return.
		if !signatureFound {
			util.Warning("%s already exists and is managed by the user.", filepath.Base(filePath))
			return nil
		}
	}

	t, err := template.New("settings.ddev.py").ParseFS(bundledAssets, path.Join("django4", "settings.ddev.py"))
	if err != nil {
		return err
	}

	// Ensure target directory exists and is writable
	dir := filepath.Dir(filePath)
	if err = os.Chmod(dir, 0755); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	settings := map[string]string{
		"host":     "db",
		"user":     "db",
		"password": "db",
		"database": "db",
		"engine":   "django.db.backends.postgresql",
	}
	if app.Database.Type == nodeps.MySQL || app.Database.Type == nodeps.MariaDB {
		settings["engine"] = "django.db.backends.mysql"
	}
	err = t.Execute(file, settings)
	if err != nil {
		return err
	}
	util.CheckClose(file)
	return nil
}
