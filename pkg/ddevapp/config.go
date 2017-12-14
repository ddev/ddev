package ddevapp

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// DefaultProviderName contains the name of the default provider which will be used if one is not otherwise specified.
const DefaultProviderName = "default"

// CurrentAppVersion sets the current YAML config file version.
// We're not doing anything with AppVersion, so just default it to 1 for now.
const CurrentAppVersion = "1"

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Command defines commands to be run as pre/post hooks
type Command struct {
	Exec     string `yaml:"exec,omitempty"`
	ExecHost string `yaml:"exec-host,omitempty"`
}

// Provider is the interface which all provider plugins must implement.
type Provider interface {
	Init(app *DdevApp) error
	ValidateField(string, string) error
	PromptForConfig() error
	Write(string) error
	Read(string) error
	Validate() error
	GetBackup(string) (fileLocation string, importPath string, err error)
}

// NewApp creates a new DdevApp struct with defaults set and overridden by any existing config.yml.
func NewApp(AppRoot string, provider string) (*DdevApp, error) {
	// Set defaults.
	app := &DdevApp{}
	app.ConfigPath = filepath.Join(AppRoot, ".ddev", "config.yaml")

	app.AppRoot = AppRoot
	app.ConfigPath = app.GetConfigPath("config.yaml")
	app.APIVersion = CurrentAppVersion

	// These should always default to the latest image/tag names from the Version package.
	app.WebImage = version.WebImg + ":" + version.WebTag
	app.DBImage = version.DBImg + ":" + version.DBTag
	app.DBAImage = version.DBAImg + ":" + version.DBATag

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	if _, err := os.Stat(app.ConfigPath); !os.IsNotExist(err) {
		err = app.ReadConfig()
		if err != nil {
			return app, fmt.Errorf("%v exists but cannot be read: %v", app.ConfigPath, err)
		}
	}

	// Allow override with "pantheon" from function provider arg, but nothing else.
	// Otherwise we accept whatever might have been in config file if there was anything.
	if provider == "" && app.Provider != "" {
		// Do nothing. This is the case where the config has a provider and no override is provided. Config wins.
	} else if provider == "pantheon" || provider == DefaultProviderName {
		app.Provider = provider // Use the provider passed-in. Function argument wins.
	} else if provider == "" && app.Provider == "" {
		app.Provider = DefaultProviderName // Nothing passed in, nothing configured. Set c.Provider to default
	} else {
		return app, fmt.Errorf("Provider '%s' is not implemented", provider)
	}

	return app, nil
}

// GetConfigPath returns the path to an application config file specified by filename.
func (app *DdevApp) GetConfigPath(filename string) string {
	return filepath.Join(app.AppRoot, ".ddev", filename)
}

// WriteConfig writes the app configuration into the .ddev folder.
func (app *DdevApp) WriteConfig() error {

	err := PrepDdevDirectory(filepath.Dir(app.ConfigPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(app)
	if err != nil {
		return err
	}

	// Append hook information and sample hook suggestions.
	cfgbytes = append(cfgbytes, []byte(HookTemplate)...)
	cfgbytes = append(cfgbytes, GetHookSuggestions(app.Type)...)

	err = ioutil.WriteFile(app.ConfigPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	return provider.Write(app.GetConfigPath("import.yaml"))
}

// ReadConfig reads app configuration from a specified location on disk, falling
// back to defaults for config values not defined in the read config file.
func (app *DdevApp) ReadConfig() error {

	source, err := ioutil.ReadFile(app.ConfigPath)
	if err != nil {
		return fmt.Errorf("could not find an active ddev configuration at %s have you run 'ddev config'? %v", app.ConfigPath, err)
	}

	// validate extend command keys
	err = validateCommandYaml(source)
	if err != nil {
		return fmt.Errorf("invalid configuration in %s: %v", app.ConfigPath, err)
	}

	// ReadConfig config values from file.
	err = yaml.Unmarshal(source, app)
	if err != nil {
		return err
	}

	// If any of these values aren't defined in the config file, set them to defaults.
	if app.Name == "" {
		app.Name = filepath.Base(app.AppRoot)
	}
	if app.WebImage == "" {
		app.WebImage = version.WebImg + ":" + version.WebTag
	}
	if app.DBImage == "" {
		app.DBImage = version.DBImg + ":" + version.DBTag
	}
	if app.DBAImage == "" {
		app.DBAImage = version.DBAImg + ":" + version.DBATag
	}

	dirPath := filepath.Join(util.GetGlobalDdevDir(), app.Name)
	app.DataDir = filepath.Join(dirPath, "mysql")
	app.ImportDir = filepath.Join(dirPath, "import-db")

	SetSiteSettingsPaths(app)

	return err
}

// WarnIfConfigReplace just messages user about whether config is being replaced or created
func (app *DdevApp) WarnIfConfigReplace() {
	if app.ConfigExists() {
		util.Warning("You are reconfiguring the app at %s. \nThe existing configuration will be updated and replaced.", app.AppRoot)
	} else {
		util.Success("Creating a new ddev project config in the current directory (%s)", app.AppRoot)
		util.Success("Once completed, your configuration will be written to %s\n", app.ConfigPath)
	}
}

// PromptForConfig goes through a set of prompts to receive user input and generate an Config struct.
func (app *DdevApp) PromptForConfig() error {

	app.WarnIfConfigReplace()

	for {
		err := app.promptForName()

		if err == nil {
			break
		}

		output.UserOut.Printf("%v", err)
	}

	for {
		err := app.docrootPrompt()

		if err == nil {
			break
		}

		output.UserOut.Printf("%v", err)
	}

	err := app.appTypePrompt()
	if err != nil {
		return err
	}

	err = app.providerInstance.PromptForConfig()

	return err
}

// ValidateConfig ensures the configuration meets ddev's requirements.
func (app *DdevApp) ValidateConfig() error {
	// validate docroot
	fullPath := filepath.Join(app.AppRoot, app.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("no directory could be found at %s. Please enter a valid docroot in your configuration", fullPath)
	}

	// validate hostname
	match := hostRegex.MatchString(app.GetHostname())
	if !match {
		return fmt.Errorf("%s is not a valid hostname. Please enter a site name in your configuration that will allow for a valid hostname. See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements", app.GetHostname())
	}

	// validate apptype
	match = IsValidAppType(app.Type)
	if !match {
		return fmt.Errorf("'%s' is not a valid apptype", app.Type)
	}

	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the
// docker-compose.yaml should exist for this app.
func (app *DdevApp) DockerComposeYAMLPath() string {
	return app.GetConfigPath("docker-compose.yaml")
}

// GetHostname returns the hostname of the app.
func (app *DdevApp) GetHostname() string {
	return app.Name + "." + version.DDevTLD
}

// WriteDockerComposeConfig writes a docker-compose.yaml to the app configuration directory.
func (app *DdevApp) WriteDockerComposeConfig() error {
	var err error

	if !fileutil.FileExists(app.DockerComposeYAMLPath()) {

		// nolint: vetshadow
		f, err := os.Create(app.DockerComposeYAMLPath())
		if err != nil {
			return err
		}
		defer util.CheckClose(f)

		rendered, err := app.RenderComposeYAML()
		if err != nil {
			return err
		}
		_, err = f.WriteString(rendered)
		if err != nil {
			return err
		}
	}
	return err
}

// RenderComposeYAML renders the contents of docker-compose.yaml.
func (app *DdevApp) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}
	templateVars := map[string]string{
		"name":        app.Name,
		"plugin":      "ddev",
		"appType":     app.Type,
		"mailhogport": appports.GetPort("mailhog"),
		"dbaport":     appports.GetPort("dba"),
		"dbport":      appports.GetPort("db"),
	}

	err = templ.Execute(&doc, templateVars)
	return doc.String(), err
}

// Define an application name.
func (app *DdevApp) promptForName() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	namePrompt := "Project name"
	if app.Name == "" {
		dir, err := os.Getwd()
		// if working directory name is invalid for hostnames, we shouldn't suggest it
		if err == nil && hostRegex.MatchString(filepath.Base(dir)) {

			app.Name = filepath.Base(dir)
		}
	}

	namePrompt = fmt.Sprintf("%s (%s)", namePrompt, app.Name)
	fmt.Print(namePrompt + ": ")
	app.Name = util.GetInput(app.Name)
	return provider.ValidateField("Name", app.Name)
}

// Determine the document root.
func (app *DdevApp) docrootPrompt() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}

	// Determine the document root.
	output.UserOut.Printf("\nThe docroot is the directory from which your site is served. This is a relative path from your application root (%s)", app.AppRoot)
	output.UserOut.Println("You may leave this value blank if your site files are in the application root")
	var docrootPrompt = "Docroot Location"
	if app.Docroot != "" {
		docrootPrompt = fmt.Sprintf("%s (%s)", docrootPrompt, app.Docroot)
	}

	fmt.Print(docrootPrompt + ": ")
	app.Docroot = util.GetInput(app.Docroot)

	// Ensure the docroot exists. If it doesn't, prompt the user to verify they entered it correctly.
	fullPath := filepath.Join(app.AppRoot, app.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		output.UserOut.Errorf("No directory could be found at %s. Please enter a valid docroot\n", fullPath)
		app.Docroot = ""
		return app.docrootPrompt()
	}
	return provider.ValidateField("Docroot", app.Docroot)
}

// ConfigExists determines if a ddev config file exists for this application.
func (app *DdevApp) ConfigExists() bool {
	if _, err := os.Stat(app.ConfigPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// appTypePrompt handles the Type workflow.
func (app *DdevApp) appTypePrompt() error {
	provider, err := app.GetProvider()
	if err != nil {
		return err
	}
	var appType string
	typePrompt := fmt.Sprintf("Application Type [%s]", strings.Join(GetValidAppTypes(), ", "))

	// First, see if we can auto detect what kind of site it is so we can set a sane default.
	absDocroot := filepath.Join(app.AppRoot, app.Docroot)
	log.WithFields(log.Fields{
		"Location": absDocroot,
	}).Debug("Attempting to auto-determine application type")

	appType, err = DetermineAppType(absDocroot)
	if err == nil {
		// If we found an application type just set it and inform the user.
		util.Success("Found a %s codebase at %s.", appType, filepath.Join(app.AppRoot, app.Docroot))
		app.Type = appType
		return provider.ValidateField("Type", app.Type)
	}
	typePrompt = fmt.Sprintf("%s (%s)", typePrompt, app.Type)

	if !IsValidAppType(appType) {
		fmt.Printf(typePrompt + ": ")
		appType = strings.ToLower(util.GetInput(app.Type))

		if !IsValidAppType(appType) {
			output.UserOut.Errorf("'%s' is not a valid application type. Allowed application types are: %s\n", appType, strings.Join(GetValidAppTypes(), ", "))
		}
		app.Type = appType
	}
	return provider.ValidateField("Type", app.Type)
}

// PrepDdevDirectory creates a .ddev directory in the current working directory
func PrepDdevDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {

		log.WithFields(log.Fields{
			"directory": dir,
		}).Debug("Config Directory does not exist, attempting to create.")

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// DetermineAppType uses some predetermined file checks to determine if an app
// is of any of the known types
func DetermineAppType(basePath string) (string, error) {
	defaultLocations := map[string]string{
		"scripts/drupal.sh":      "drupal7",
		"core/scripts/drupal.sh": "drupal8",
		"wp-settings.php":        "wordpress",
	}

	for k, v := range defaultLocations {
		fp := filepath.Join(basePath, k)
		log.WithFields(log.Fields{
			"file": fp,
		}).Debug("Looking for app fingerprint.")
		if _, err := os.Stat(fp); err == nil {
			log.WithFields(log.Fields{
				"file": fp,
				"app":  v,
			}).Debug("Found app fingerprint.")

			return v, nil
		}
	}

	return "", errors.New("DetermineAppType() couldn't determine app's type")
}

// validateCommandYaml validates command hooks and tasks defined in hooks for config.yaml
func validateCommandYaml(source []byte) error {
	validHooks := []string{
		"pre-start",
		"post-start",
		"pre-import-db",
		"post-import-db",
		"pre-import-files",
		"post-import-files",
	}

	validTasks := []string{
		"exec",
		"exec-host",
	}

	type Validate struct {
		Commands map[string][]map[string]interface{} `yaml:"hooks,omitempty"`
	}
	val := &Validate{}

	err := yaml.Unmarshal(source, val)
	if err != nil {
		return err
	}

	for command, tasks := range val.Commands {
		var match bool
		for _, hook := range validHooks {
			if command == hook {
				match = true
			}
		}
		if !match {
			return fmt.Errorf("invalid command hook %s defined in config.yaml", command)
		}

		for _, taskSet := range tasks {
			for taskName := range taskSet {
				var match bool
				for _, validTask := range validTasks {
					if taskName == validTask {
						match = true
					}
				}
				if !match {
					return fmt.Errorf("invalid task '%s' defined for %s hook in config.yaml", taskName, command)
				}
			}
		}

	}

	return nil
}
