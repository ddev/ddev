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

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/drud/ddev/pkg/appports"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// CurrentAppVersion sets the current YAML config file version.
// We're not doing anything with AppVersion, so just default it to 1 for now.
const CurrentAppVersion = "1"

// DDevDefaultPlatform defines the DDev Platform. It's just hardcoded for now, but should be adjusted as we add more platforms.
const DDevDefaultPlatform = "local"

// DDevTLD defines the tld to use for DDev site URLs.
const DDevTLD = "ddev.local"

// AllowedAppTypes lists the types of site/app that can be used.
var AllowedAppTypes = []string{"drupal7", "drupal8", "wordpress"}

// Regexp pattern to determine if a hostname is valid per RFC 1123.
var hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)

// Config defines the yaml config file format for ddev applications
type Config struct {
	APIVersion string `yaml:"APIVersion"`
	Name       string `yaml:"name"`
	AppType    string `yaml:"type"`
	Docroot    string `yaml:"docroot"`
	WebImage   string `yaml:"webimage"`
	DBImage    string `yaml:"dbimage"`
	DBAImage   string `yaml:"dbaimage"`
	ConfigPath string `yaml:"-"`
	AppRoot    string `yaml:"-"`
	Platform   string `yaml:"-"`
	DataDir    string `yaml:"-"`
	ImportDir  string `yaml:"-"`
}

// NewConfig creates a new Config struct with defaults set. It is preferred to using new() directly.
func NewConfig(AppRoot string) (*Config, error) {
	// Set defaults.
	c := &Config{}
	c.ConfigPath = filepath.Join(AppRoot, ".ddev", "config.yaml")
	c.AppRoot = AppRoot
	c.APIVersion = CurrentAppVersion

	// Default platform for now.
	c.Platform = DDevDefaultPlatform

	// These should always default to the latest image/tag names from the Version package.
	c.WebImage = version.WebImg + ":" + version.WebTag
	c.DBImage = version.DBImg + ":" + version.DBTag
	c.DBAImage = version.DBAImg + ":" + version.DBATag

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	err := c.Read()
	if err != nil {
		return c, err
	}

	return c, nil
}

// Write the app configuration to the .ddev folder.
func (c *Config) Write() error {
	err := PrepDdevDirectory(filepath.Dir(c.ConfigPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"location": c.ConfigPath,
	}).Debug("Writing Config")
	err = ioutil.WriteFile(c.ConfigPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Read app configuration from a specified location on disk, falling back to defaults for config
// values not defined in the read config file.
func (c *Config) Read() error {
	source, err := ioutil.ReadFile(c.ConfigPath)
	if err != nil {
		return err
	}

	// Read config values from file.
	err = yaml.Unmarshal(source, c)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"Read config": awsutil.Prettify(c),
	}).Debug("Finished config read")

	// If any of these values aren't defined in the config file, set them to defaults.
	if c.Name == "" {
		c.Name = filepath.Base(c.AppRoot)
	}
	if c.WebImage == "" {
		c.WebImage = version.WebImg + ":" + version.WebTag
	}
	if c.DBImage == "" {
		c.DBImage = version.DBImg + ":" + version.DBTag
	}
	if c.DBAImage == "" {
		c.DBAImage = version.DBAImg + ":" + version.DBATag
	}

	log.WithFields(log.Fields{
		"Active config": awsutil.Prettify(c),
	}).Debug("Finished config set")
	return nil
}

// Config goes through a set of prompts to receive user input and generate an Config struct.
func (c *Config) Config() error {

	if c.ConfigExists() {
		fmt.Printf("Editing existing ddev project at %s\n\n", c.AppRoot)
	} else {
		fmt.Printf("Creating a new ddev project config in the current directory (%s)\n", c.AppRoot)
		fmt.Printf("Once completed, your configuration will be written to %s\n\n\n", c.ConfigPath)
	}

	// Log what the starting config is, for debugging purposes.
	log.WithFields(log.Fields{
		"Existing config": awsutil.Prettify(c),
	}).Debug("Configuring application")

	err := c.namePrompt()
	if err != nil {
		return err
	}

	err = c.docrootPrompt()
	if err != nil {
		return err
	}

	err = c.appTypePrompt()
	if err != nil {
		return err
	}

	// Log the resulting config, for debugging purposes.
	log.WithFields(log.Fields{
		"Config": awsutil.Prettify(c),
	}).Debug("Configuration completed")

	return nil
}

// Validate ensures the configuraton meets ddev's requirements.
func (c *Config) Validate() error {
	// validate docroot
	fullPath := filepath.Join(c.AppRoot, c.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("no directory could be found at %s. Please enter a valid docroot in your configuration", fullPath)
	}

	// validate hostname
	match := hostRegex.MatchString(c.Hostname())
	if !match {
		return fmt.Errorf("%s is not a valid hostname. Please enter a site name in your configuration that will allow for a valid hostname. See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements", c.Hostname())
	}

	// validate apptype
	match = IsAllowedAppType(c.AppType)
	if !match {
		return fmt.Errorf("%s is not a valid apptype", c.AppType)
	}

	return nil
}

// DockerComposeYAMLPath returns the absolute path to where the docker-compose.yaml should exist for this app configuration.
func (c *Config) DockerComposeYAMLPath() string {
	return filepath.Join(c.AppRoot, ".ddev", "docker-compose.yaml")
}

// Hostname returns the hostname to the app controlled by this config.
func (c *Config) Hostname() string {
	return c.Name + "." + DDevTLD
}

// WriteDockerComposeConfig writes a docker-compose.yaml to the app configuration directory.
func (c *Config) WriteDockerComposeConfig() error {
	log.WithFields(log.Fields{
		"Location": c.DockerComposeYAMLPath(),
	}).Debug("Writing docker-compose.yaml")

	f, err := os.Create(c.DockerComposeYAMLPath())
	if err != nil {
		return err
	}
	defer util.CheckClose(f)

	rendered, err := c.RenderComposeYAML()
	if err != nil {
		return err
	}
	_, err = f.WriteString(rendered)

	return err
}

// RenderComposeYAML renders the contents of docker-compose.yaml.
func (c *Config) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}
	templateVars := map[string]string{
		"name":        c.Name,
		"plugin":      "ddev",
		"appType":     c.AppType,
		"mailhogport": appports.GetPort("mailhog"),
		"dbaport":     appports.GetPort("dba"),
		"dbport":      appports.GetPort("db"),
	}

	err = templ.Execute(&doc, templateVars)
	return doc.String(), err
}

// Define an application name.
func (c *Config) namePrompt() error {
	namePrompt := "Project name"
	if c.Name == "" {
		dir, err := os.Getwd()
		// if working directory name is invalid for hostnames, we shouldn't suggest it
		if err == nil && hostRegex.MatchString(filepath.Base(dir)) {
			c.Name = filepath.Base(dir)
		}
	}

	namePrompt = fmt.Sprintf("%s (%s)", namePrompt, c.Name)
	fmt.Print(namePrompt + ": ")
	c.Name = util.GetInput(c.Name)

	match := hostRegex.MatchString(c.Hostname())
	if !match {
		fmt.Printf("%s is not a valid hostname. Please enter a site name that will allow for a valid hostname.\n See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_hostnames for valid hostname requirements \n", c.Hostname())
		c.Name = ""
		return c.namePrompt()
	}

	return nil
}

// Determine the document root.
func (c *Config) docrootPrompt() error {
	fmt.Printf("\nThe docroot is the directory from which your site is served. This is a relative path from your application root (%s)\n", c.AppRoot)
	fmt.Println("You may leave this value blank if your site files are in the application root")
	var docrootPrompt = "Docroot Location"
	if c.Docroot != "" {
		docrootPrompt = fmt.Sprintf("%s (%s)", docrootPrompt, c.Docroot)
	}

	fmt.Print(docrootPrompt + ": ")
	c.Docroot = util.GetInput(c.Docroot)

	// Ensure the docroot exists. If it doesn't, prompt the user to verify they entered it correctly.
	fullPath := filepath.Join(c.AppRoot, c.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("No directory could be found at %s. Please enter a valid docroot", fullPath)
		c.Docroot = ""
		return c.docrootPrompt()
	}
	return nil
}

// ConfigExists determines if a ddev config file exists for this application.
func (c *Config) ConfigExists() bool {
	if _, err := os.Stat(c.ConfigPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// appTypePrompt handles the AppType workflow.
func (c *Config) appTypePrompt() error {
	var appType string
	typePrompt := fmt.Sprintf("Application Type [%s]", strings.Join(AllowedAppTypes, ", "))

	// First, see if we can auto detect what kind of site it is so we can set a sane default.
	absDocroot := filepath.Join(c.AppRoot, c.Docroot)
	log.WithFields(log.Fields{
		"Location": absDocroot,
	}).Debug("Attempting to auto-determine application type")

	appType, err := determineAppType(absDocroot)
	if err == nil {
		// If we found an application type just set it and inform the user.
		fmt.Printf("Found a %s codebase at %s\n", appType, filepath.Join(c.AppRoot, c.Docroot))
		c.AppType = appType
		return nil
	}
	typePrompt = fmt.Sprintf("%s (%s)", typePrompt, c.AppType)

	for IsAllowedAppType(appType) != true {
		fmt.Printf(typePrompt + ": ")
		appType = strings.ToLower(util.GetInput(c.AppType))

		if IsAllowedAppType(appType) != true {
			fmt.Printf("%s is not a valid application type. Allowed application types are: %s\n", appType, strings.Join(AllowedAppTypes, ", "))
		}
		c.AppType = appType
	}
	return nil
}

// IsAllowedAppType determines if a given string exists in the AllowedAppTypes slice.
func IsAllowedAppType(appType string) bool {
	for _, t := range AllowedAppTypes {
		if appType == t {
			return true
		}
	}
	return false
}

// PrepDdevDirectory creates a .ddev directory in the current working
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

// DetermineAppType uses some predetermined file checks to determine if a local app
// is of any of the known types
func determineAppType(basePath string) (string, error) {
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

	return "", errors.New("determineAppType() couldn't determine app's type")
}
