package ddevapp

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// LooksLikeWebProject reports whether app.AppRoot holds code DDEV could serve:
// a codebase one of the apptype detectors recognizes, a Composer or npm
// project, or an index file at the root or in a common docroot.
func (app *DdevApp) LooksLikeWebProject() bool {
	if app.detectSpecificAppType() != "" {
		return true
	}
	for _, f := range []string{"composer.json", "package.json"} {
		if fileutil.FileExists(filepath.Join(app.AppRoot, f)) {
			return true
		}
	}
	for _, dir := range append([]string{"."}, AvailablePHPDocrootLocations()...) {
		for _, f := range []string{"index.php", "index.html"} {
			if fileutil.FileExists(filepath.Join(app.AppRoot, dir, f)) {
				return true
			}
		}
	}
	return false
}

// NewAutoConfigApp returns the project `ddev config --auto` would configure
// in appRoot, without writing anything.
func NewAutoConfigApp(appRoot string) (*DdevApp, error) {
	app, err := NewApp(appRoot, false)
	if err != nil {
		return nil, err
	}
	if app.Name == "" {
		app.Name = NormalizeProjectName(filepath.Base(appRoot))
	}
	app.Docroot = DiscoverDefaultDocroot(app)
	app.Type = app.DetectAppType()
	if err := app.ConfigFileOverrideAction(false); err != nil {
		return nil, err
	}
	app.SetApptypeSettingsPaths()
	return app, nil
}

// AutoConfigProblem returns why the configuration from NewAutoConfigApp
// can't be used as is, or "" when it can.
func (app *DdevApp) AutoConfigProblem() string {
	if err := ValidateProjectName(app.Name); err != nil {
		return fmt.Sprintf("'%s' is not a valid project name.", app.Name)
	}
	if p := globalconfig.GetProject(app.Name); p != nil && p.AppRoot != "" && filepath.Clean(p.AppRoot) != filepath.Clean(app.AppRoot) {
		return fmt.Sprintf("The project name '%s' is already used by the project in %s.", app.Name, p.AppRoot)
	}
	return ""
}

// AutoConfigSummary describes the configuration from NewAutoConfigApp.
func (app *DdevApp) AutoConfigSummary() string {
	docroot := app.Docroot
	if docroot == "" {
		docroot = "(project root)"
	}
	url := app.GetPrimaryURL()
	if url == "" {
		url = "https://" + app.GetHostname()
	}
	lines := []string{
		fmt.Sprintf("  Name:      %s", app.Name),
		fmt.Sprintf("  URL:       %s", url),
		fmt.Sprintf("  Type:      %s", app.Type),
		fmt.Sprintf("  Docroot:   %s", docroot),
		fmt.Sprintf("  PHP:       %s", app.PHPVersion),
		fmt.Sprintf("  Database:  %s:%s", app.Database.Type, app.Database.Version),
		fmt.Sprintf("  Webserver: %s", app.WebserverType),
		"",
		"This creates .ddev/config.yaml" + app.settingsFilesSummary() + ".",
	}
	if globalconfig.DdevGlobalConfig.OmitProjectNameByDefault {
		lines = append(lines, "The name is not written to config.yaml because omit_project_name_by_default is set.")
	}
	return strings.Join(lines, "\n")
}

// settingsFilesSummary names the CMS settings files that configuring the
// project creates or changes.
func (app *DdevApp) settingsFilesSummary() string {
	appFuncs, ok := appTypeMatrix[app.Type]
	if !ok || appFuncs.settingsCreator == nil || app.DisableSettingsManagement {
		return ""
	}
	var files []string
	for _, f := range []string{app.SiteDdevSettingsFile, app.SiteSettingsPath} {
		if f == "" {
			continue
		}
		if rel, err := filepath.Rel(app.AppRoot, f); err == nil {
			f = rel
		}
		if !slices.Contains(files, f) {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return ""
	}
	return ", and DDEV may create or edit " + strings.Join(files, " and ")
}
