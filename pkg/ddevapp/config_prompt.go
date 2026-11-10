package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// PromptForConfig shows the configuration DDEV detects for the project and
// asks whether to use it. If the answer is no, or the detected configuration
// can't be used, it asks about each setting as PromptForCustomConfig does.
func (app *DdevApp) PromptForConfig() error {
	app.WarnIfConfigReplace()

	// The questions start from the app as it was, not from the suggestion.
	orig := *app
	if err := app.applySuggestedConfig(); err != nil {
		return err
	}
	heading := "DDEV suggests this configuration:"
	if app.ConfigExists() {
		heading = "DDEV suggests keeping this configuration:"
	}
	output.UserOut.Printf("%s\n\n%s\n\n", heading, app.AutoConfigSummary())

	if problem := app.AutoConfigProblem(); problem != "" {
		util.Warning("%s", problem)
	} else if confirmSuggestedConfig() {
		if err := app.CreateDocroot(); err != nil {
			return err
		}
		return app.ValidateConfig()
	}

	*app = orig
	return app.promptForSettings()
}

// PromptForCustomConfig asks for each of the main settings in turn, offering
// the detected value as the default, so that <RETURN> accepts it.
func (app *DdevApp) PromptForCustomConfig() error {
	app.WarnIfConfigReplace()
	return app.promptForSettings()
}

// promptForSettings asks the questions. The project type comes before the
// settings that its ConfigFileOverrideAction provides defaults for.
func (app *DdevApp) promptForSettings() error {
	if err := app.projectNamePrompt(); err != nil {
		return err
	}
	if err := app.docrootPrompt(); err != nil {
		return err
	}
	if err := app.AppTypePrompt(); err != nil {
		return err
	}
	if err := app.ConfigFileOverrideAction(false); err != nil {
		return err
	}
	app.phpVersionPrompt()
	app.databasePrompt()
	app.webserverTypePrompt()

	return app.ValidateConfig()
}

// applySuggestedConfig sets the name, docroot, and project type DDEV detects,
// and the settings that project type calls for, as `ddev config --auto` does.
func (app *DdevApp) applySuggestedConfig() error {
	app.Name = app.defaultProjectName()
	app.Docroot = DiscoverDefaultDocroot(app)
	app.Type = app.defaultAppType()
	return app.ConfigFileOverrideAction(false)
}

// confirmSuggestedConfig asks whether to use the configuration just shown.
func confirmSuggestedConfig() bool {
	for {
		switch strings.ToLower(util.Prompt("Use this configuration? Answer 'no' to choose each setting [Y/n]", "yes")) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}

// defaultProjectName returns app.Name, or a name based on the current
// directory if there is none and the directory name can be made into one.
func (app *DdevApp) defaultProjectName() string {
	if app.Name != "" {
		return app.Name
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	name := NormalizeProjectName(filepath.Base(dir))
	if !hostRegex.MatchString(name) {
		return ""
	}
	return name
}

// defaultAppType returns the configured project type if there is a valid one,
// and otherwise the type detected in the codebase.
func (app *DdevApp) defaultAppType() string {
	if app.Type != nodeps.AppTypeNone && IsValidAppType(app.Type) {
		return app.Type
	}
	return app.DetectAppType()
}

// projectNamePrompt asks for a project name that is valid and not already
// used by a project in another directory.
func (app *DdevApp) projectNamePrompt() error {
	app.Name = app.defaultProjectName()

	for {
		fmt.Printf("Project name (%s): ", app.Name)
		name, ok := util.GetInputOK(app.Name)
		err := ValidateProjectName(name)
		if err == nil {
			app.Name = name
			problem := app.AutoConfigProblem()
			if problem == "" {
				break
			}
			err = fmt.Errorf("%s", problem)
		}
		if !ok {
			return err
		}
		output.UserOut.Print(util.ColorizeText(err.Error(), "yellow"))
	}

	return app.CheckExistingAppInApproot()
}

// docrootPrompt asks for the docroot, and creates it if it doesn't exist.
func (app *DdevApp) docrootPrompt() error {
	output.UserOut.Printf("\nThe docroot is the directory from which your site is served.\nThis is a relative path from your project root at %s\n", app.AppRoot)
	output.UserOut.Printf("Leave docroot empty (hit <RETURN>) to use the location shown in parentheses.\nOr specify a custom path if your index.php is in a different directory.\nOr use '.' (a dot) to explicitly set it to the project root.\n")
	defaultDocroot := DiscoverDefaultDocroot(app)
	shownDefault := defaultDocroot
	if shownDefault == "" {
		shownDefault = "project root"
	}

	for {
		fmt.Printf("Docroot Location (%s): ", shownDefault)
		input, ok := util.GetInputOK(defaultDocroot)
		app.Docroot = strings.Trim(input, `"' `)
		if app.Docroot == "" {
			app.Docroot = defaultDocroot
		}

		err := app.CreateDocroot()
		if err == nil {
			break
		}
		if !ok {
			return err
		}
		output.UserOut.Print(util.ColorizeText(err.Error(), "yellow"))
	}
	output.UserOut.Println()

	return nil
}

// AppTypePrompt asks for the project type, defaulting to the configured or
// detected one.
func (app *DdevApp) AppTypePrompt() error {
	util.Success("Found a %s codebase at %s.", app.DetectAppType(), app.GetAbsDocroot(false))

	app.Type = util.PromptChoice("project type", GetValidAppTypes(), app.defaultAppType())

	return nil
}

// phpVersionPrompt asks for the PHP version.
func (app *DdevApp) phpVersionPrompt() {
	defaultVersion := app.PHPVersion
	if !nodeps.IsValidPHPVersion(defaultVersion) {
		defaultVersion = nodeps.PHPDefault
	}
	app.PHPVersion = util.PromptChoice("PHP version", nodeps.GetValidPHPVersions(), defaultVersion)
}

// databasePrompt asks for the database type and then for one of that type's
// versions, unless the project has no db container.
func (app *DdevApp) databasePrompt() {
	if slices.Contains(app.GetOmittedContainers(), "db") {
		return
	}
	versions := map[string][]string{
		nodeps.MariaDB:  nodeps.GetValidMariaDBVersions(),
		nodeps.MySQL:    nodeps.GetValidMySQLVersions(),
		nodeps.Postgres: nodeps.GetValidPostgresVersions(),
	}
	defaultVersions := map[string]string{
		nodeps.MariaDB:  nodeps.MariaDBDefaultVersion,
		nodeps.MySQL:    nodeps.MySQL84,
		nodeps.Postgres: nodeps.PostgresDefaultVersion,
	}
	names := []string{nodeps.MariaDB, nodeps.MySQL, nodeps.Postgres}
	displayNames := map[string]string{nodeps.MariaDB: "MariaDB", nodeps.MySQL: "MySQL", nodeps.Postgres: "PostgreSQL"}

	orig := app.Database
	defaultType := orig.Type
	if !slices.Contains(names, defaultType) {
		defaultType = nodeps.MariaDB
	}
	dbType := util.PromptChoice("database type", names, defaultType)

	defaultVersion := defaultVersions[dbType]
	if dbType == orig.Type && nodeps.IsValidDatabaseVersion(dbType, orig.Version) {
		defaultVersion = orig.Version
	}
	app.Database = DatabaseDesc{
		Type:    dbType,
		Version: util.PromptChoice(displayNames[dbType]+" version", versions[dbType], defaultVersion),
	}

	// A database volume can only be left over from an earlier configuration.
	if app.Database == orig || !app.ConfigExists() {
		return
	}
	existing, err := app.GetExistingDBType()
	if err == nil && existing != "" && existing != app.Database.Type+":"+app.Database.Version {
		util.Warning("This %s", app.DatabaseMismatchMessage(existing))
	}
}

// webserverTypePrompt asks for the webserver type.
func (app *DdevApp) webserverTypePrompt() {
	types := nodeps.GetValidWebserverTypes()
	sort.Strings(types)
	defaultType := app.WebserverType
	if !nodeps.IsValidWebserverType(defaultType) {
		defaultType = nodeps.WebserverDefault
	}
	app.WebserverType = util.PromptChoice("webserver type", types, defaultType)
}
