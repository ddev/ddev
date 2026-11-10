package ddevapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// ConfigChoices are the settings interactive `ddev config` asks for.
type ConfigChoices struct {
	Name          string
	Docroot       string
	Type          string
	PHPVersion    string
	Database      DatabaseDesc
	WebserverType string
}

// SuggestedConfigChoices returns the name, docroot, and project type DDEV
// detects, and the settings that project type calls for, as `ddev config --auto` would use.
func (app *DdevApp) SuggestedConfigChoices() ConfigChoices {
	c := ConfigChoices{Name: app.defaultProjectName(), Docroot: DiscoverDefaultDocroot(app)}
	// Some apptype detectors look in the docroot.
	withDocroot := *app
	withDocroot.Docroot = c.Docroot
	c.Type = withDocroot.defaultAppType()
	return app.ConfigTypeDefaults(c)
}

// ConfigTypeDefaults returns c with the PHP version, database, and webserver
// type that project type c.Type calls for in a new project, or app's own
// where it calls for none or app is already configured. A project type that
// needs a docroot, like wp-bedrock, also sets it if c has none. Unlike
// ConfigFileOverrideAction, it reports nothing and changes only c.
func (app *DdevApp) ConfigTypeDefaults(c ConfigChoices) ConfigChoices {
	c.PHPVersion, c.Database, c.WebserverType = app.PHPVersion, app.Database, app.WebserverType
	preview := app.withChoices(c)
	if appFuncs, ok := appTypeMatrix[c.Type]; ok && appFuncs.configOverrideAction != nil && !app.ConfigExists() {
		_ = appFuncs.configOverrideAction(preview)
	}
	c.Docroot, c.PHPVersion, c.Database, c.WebserverType = preview.Docroot, preview.PHPVersion, preview.Database, preview.WebserverType
	return c
}

// withChoices returns a copy of app that has the settings in c.
func (app *DdevApp) withChoices(c ConfigChoices) *DdevApp {
	preview := *app
	preview.Name, preview.Docroot, preview.Type = c.Name, c.Docroot, c.Type
	preview.PHPVersion, preview.Database, preview.WebserverType = c.PHPVersion, c.Database, c.WebserverType
	return &preview
}

// ConfigChoicesSummary describes app's configuration with the settings in c.
func (app *DdevApp) ConfigChoicesSummary(c ConfigChoices) string {
	return app.withChoices(c).AutoConfigSummary()
}

// ValidateConfigChoices returns what is wrong with the settings in c, if anything.
func (app *DdevApp) ValidateConfigChoices(c ConfigChoices) error {
	if err := ValidateProjectName(c.Name); err != nil {
		return fmt.Errorf("'%s' is not a valid project name; use letters, digits, hyphens, and dots", c.Name)
	}
	if problem := app.withChoices(c).AutoConfigProblem(); problem != "" {
		return errors.New(problem)
	}
	if err := ValidateDocroot(c.Docroot); err != nil {
		return err
	}
	if !IsValidAppType(c.Type) {
		return fmt.Errorf("'%s' is not a valid project type", c.Type)
	}
	if !nodeps.IsValidPHPVersion(c.PHPVersion) {
		return fmt.Errorf("'%s' is not a valid PHP version", c.PHPVersion)
	}
	if !nodeps.IsValidDatabaseVersion(c.Database.Type, c.Database.Version) {
		return fmt.Errorf("'%s:%s' is not a valid database", c.Database.Type, c.Database.Version)
	}
	if !nodeps.IsValidWebserverType(c.WebserverType) {
		return fmt.Errorf("'%s' is not a valid webserver type", c.WebserverType)
	}
	return nil
}

// ApplyConfigChoices configures app with the settings in c, creating the
// docroot if needed. A database is set only if it isn't the one the project
// type calls for, which ConfigFileOverrideAction sets unless a database exists.
func (app *DdevApp) ApplyConfigChoices(c ConfigChoices) error {
	if err := app.ValidateConfigChoices(c); err != nil {
		return err
	}
	typeDatabase := app.ConfigTypeDefaults(c).Database

	app.Name = c.Name
	if err := app.CheckExistingAppInApproot(); err != nil {
		return err
	}
	app.Docroot = c.Docroot
	if err := app.CreateDocroot(); err != nil {
		return err
	}
	app.Type = c.Type
	if err := app.ConfigFileOverrideAction(false); err != nil {
		return err
	}
	app.PHPVersion, app.WebserverType = c.PHPVersion, c.WebserverType
	if c.Database != typeDatabase && c.Database != app.Database {
		app.Database = c.Database
		app.warnIfDatabaseMismatch()
	}

	return app.ValidateConfig()
}

// projectTypeFamily is a group of project types that interactive `ddev config`
// lists as one, like drupal for drupal6 to drupal12, followed by a choice
// of which of them.
type projectTypeFamily struct {
	name string
	// label names the choice among the types, like "Drupal version".
	label string
	// options name the types, which types holds in the same order, and
	// display, if set, is how the form shows them.
	options []string
	types   []string
	display []string
	// fallback is the option offered when the codebase shows no other.
	fallback string
}

// projectTypeFamilies returns the groups of project types.
func projectTypeFamilies() []projectTypeFamily {
	drupal := projectTypeFamily{name: nodeps.AppTypeDrupal, label: "Drupal version"}
	for _, t := range GetValidAppTypes() {
		if v, ok := strings.CutPrefix(t, nodeps.AppTypeDrupal); ok && v != "" && strings.Trim(v, "0123456789") == "" {
			drupal.options = append(drupal.options, v)
		}
	}
	slices.SortFunc(drupal.options, func(a, b string) int {
		x, _ := strconv.Atoi(a)
		y, _ := strconv.Atoi(b)
		return x - y
	})
	for _, v := range drupal.options {
		drupal.types = append(drupal.types, nodeps.AppTypeDrupal+v)
	}
	drupal.fallback = strings.TrimPrefix(nodeps.AppTypeDrupalLatestStable, nodeps.AppTypeDrupal)

	return []projectTypeFamily{
		drupal,
		{
			name: nodeps.AppTypeMagento, label: "Magento version",
			options:  []string{"1", "2"},
			types:    []string{nodeps.AppTypeMagento, nodeps.AppTypeMagento2},
			display:  []string{"1 / OpenMage", "2"},
			fallback: "2",
		},
		{
			name: nodeps.AppTypeWordPress, label: "WordPress variant",
			options:  []string{"standard", "bedrock"},
			types:    []string{nodeps.AppTypeWordPress, nodeps.AppTypeWPBedrock},
			fallback: "standard",
		},
	}
}

// ProjectTypeFamilies returns the valid project types with each group of
// related types, like drupal6 to drupal12, listed once by its family name.
func ProjectTypeFamilies() []string {
	var families []string
	for _, t := range GetValidAppTypes() {
		if family, _ := SplitProjectType(t); !slices.Contains(families, family) {
			families = append(families, family)
		}
	}
	return families
}

// SplitProjectType returns the family and option of a project type in a
// group, like drupal and 11 for drupal11 or wordpress and bedrock for
// wp-bedrock, and appType and "" for any other.
func SplitProjectType(appType string) (family, option string) {
	for _, f := range projectTypeFamilies() {
		if i := slices.Index(f.types, appType); i >= 0 {
			return f.name, f.options[i]
		}
	}
	return appType, ""
}

// JoinProjectType returns the project type of family's option, as
// SplitProjectType splits it, or family if it has no such option.
func JoinProjectType(family, option string) string {
	for _, f := range projectTypeFamilies() {
		if i := slices.Index(f.options, option); f.name == family && i >= 0 {
			return f.types[i]
		}
	}
	return family
}

// ProjectTypeVersions returns the options family comes in, and the one
// offered when the codebase shows no other, or nil if it is not a family.
func ProjectTypeVersions(family string) ([]string, string) {
	for _, f := range projectTypeFamilies() {
		if f.name == family {
			return f.options, f.fallback
		}
	}
	return nil, ""
}

// ProjectTypeOptionDisplay returns how family's option is shown, like
// "1 / OpenMage" for Magento 1.
func ProjectTypeOptionDisplay(family, option string) string {
	for _, f := range projectTypeFamilies() {
		if i := slices.Index(f.options, option); f.name == family && i >= 0 && f.display != nil {
			return f.display[i]
		}
	}
	return option
}

// ProjectTypeVersionLabel names the choice among family's options, like
// "Drupal version".
func ProjectTypeVersionLabel(family string) string {
	for _, f := range projectTypeFamilies() {
		if f.name == family {
			return f.label
		}
	}
	return family + " version"
}

// DatabaseTypes returns the database types in the order they are offered.
func DatabaseTypes() []string {
	return []string{nodeps.MariaDB, nodeps.MySQL, nodeps.Postgres}
}

// DatabaseVersions returns the valid versions of dbType and the one offered
// by default.
func DatabaseVersions(dbType string) ([]string, string) {
	switch dbType {
	case nodeps.MySQL:
		return nodeps.GetValidMySQLVersions(), nodeps.MySQL84
	case nodeps.Postgres:
		return nodeps.GetValidPostgresVersions(), nodeps.PostgresDefaultVersion
	}
	return nodeps.GetValidMariaDBVersions(), nodeps.MariaDBDefaultVersion
}

// DatabaseDisplayName returns the name dbType is shown with, like PostgreSQL.
func DatabaseDisplayName(dbType string) string {
	switch dbType {
	case nodeps.MariaDB:
		return "MariaDB"
	case nodeps.MySQL:
		return "MySQL"
	case nodeps.Postgres:
		return "PostgreSQL"
	}
	return dbType
}

// warnIfDatabaseMismatch warns if a database volume left by an earlier
// configuration holds another database than the one configured.
func (app *DdevApp) warnIfDatabaseMismatch() {
	if !app.ConfigExists() {
		return
	}
	existing, err := app.GetExistingDBType()
	if err == nil && existing != "" && existing != app.Database.Type+":"+app.Database.Version {
		util.Warning("This %s", app.DatabaseMismatchMessage(existing))
	}
}

// PromptForConfig shows the configuration DDEV detects for the project and
// asks whether to use it. If the answer is no, or the detected configuration
// can't be used, it asks about each setting as PromptForCustomConfig does.
func (app *DdevApp) PromptForConfig() error {
	app.WarnIfConfigReplace()

	suggested := app.SuggestedConfigChoices()
	heading := "DDEV suggests this configuration:"
	if app.ConfigExists() {
		heading = "DDEV suggests keeping this configuration:"
	}
	output.UserOut.Printf("%s\n\n%s\n\n", heading, app.ConfigChoicesSummary(suggested))

	if err := app.ValidateConfigChoices(suggested); err != nil {
		util.Warning("%v", err)
	} else if confirmSuggestedConfig() {
		return app.ApplyConfigChoices(suggested)
	}

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
// detected one, and then for which type of a family like Drupal or Magento.
// A full type like drupal10 or wp-bedrock is accepted as the answer too.
func (app *DdevApp) AppTypePrompt() error {
	util.Success("Found a %s codebase at %s.", app.DetectAppType(), app.GetAbsDocroot(false))

	defaultType := app.defaultAppType()
	defaultFamily, defaultVersion := SplitProjectType(defaultType)
	answer := util.PromptChoiceFrom("project type", ProjectTypeFamilies(), GetValidAppTypes(), defaultFamily)
	app.Type = answer
	if versions, latest := ProjectTypeVersions(answer); versions != nil {
		if defaultFamily != answer || defaultVersion == "" {
			defaultVersion = latest
		}
		version := util.PromptChoice(ProjectTypeVersionLabel(answer), versions, defaultVersion)
		app.Type = JoinProjectType(answer, version)
	}

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
	orig := app.Database
	defaultType := orig.Type
	if !slices.Contains(DatabaseTypes(), defaultType) {
		defaultType = nodeps.MariaDB
	}
	dbType := util.PromptChoice("database type", DatabaseTypes(), defaultType)

	versions, defaultVersion := DatabaseVersions(dbType)
	if dbType == orig.Type && nodeps.IsValidDatabaseVersion(dbType, orig.Version) {
		defaultVersion = orig.Version
	}
	app.Database = DatabaseDesc{
		Type:    dbType,
		Version: util.PromptChoice(DatabaseDisplayName(dbType)+" version", versions, defaultVersion),
	}
	if app.Database != orig {
		app.warnIfDatabaseMismatch()
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
