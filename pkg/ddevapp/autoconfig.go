package ddevapp

import (
	"fmt"
	"path/filepath"
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

// AutoConfigProblem returns why the configuration `ddev config --auto` would
// write can't be used as is, or "" when it can.
func (app *DdevApp) AutoConfigProblem() string {
	if err := ValidateProjectName(app.Name); err != nil {
		return fmt.Sprintf("'%s' is not a valid project name.", app.Name)
	}
	if p := globalconfig.GetProject(app.Name); p != nil && p.AppRoot != "" && filepath.Clean(p.AppRoot) != filepath.Clean(app.AppRoot) {
		return fmt.Sprintf("The project name '%s' is already used by the project in %s.", app.Name, p.AppRoot)
	}
	return ""
}

// AutoConfigSummary describes the configuration `ddev config --auto` would write.
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
		"This creates .ddev/config.yaml.",
	}
	if globalconfig.DdevGlobalConfig.OmitProjectNameByDefault {
		lines = append(lines, "The name is not written to config.yaml because omit_project_name_by_default is set.")
	}
	return strings.Join(lines, "\n")
}
