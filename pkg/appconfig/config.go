package appconfig

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/pretty"
	"github.com/drud/drud-go/utils/system"
	yaml "gopkg.in/yaml.v2"
)

// CurrentAppVersion sets the current YAML config file version.
// We're not doing anything with AppVersion, so just default it to 1 for now.
const CurrentAppVersion = "1"
const DDevTLD = "ddev.local"

var allowedAppTypes = []string{"drupal7", "drupal8", "wordpress"}

// AppConfig defines the yaml config file format for ddev applications
type AppConfig struct {
	APIVersion string `yaml:"APIVersion"`
	Name       string `yaml:"name"`
	AppType    string `yaml:"type"`
	Docroot    string `yaml:"docroot"`
	WebImage   string `yaml:"webimage"`
	DBImage    string `yaml:"dbimage"`
	ConfigPath string
	AppRoot    string
}

// NewAppConfig creates a new AppConfig struct with defaults set. It is preferred to using new() directly.
func NewAppConfig(AppRoot string, ConfigPath string) (*AppConfig, error) {
	// Set defaults.
	c := &AppConfig{}
	c.ConfigPath = ConfigPath
	c.AppRoot = AppRoot
	c.APIVersion = CurrentAppVersion

	// These should always default to the latest image/tag names from the Version package.
	c.WebImage = version.WebImg + ":" + version.WebTag
	c.DBImage = version.DBImg + ":" + version.DBTag

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	if c.configExists() {
		err := c.Read()
		if err != nil {
			return c, err
		}
	}

	return c, nil
}

// Write the app configuration to a specific location on disk
func (c *AppConfig) Write() error {
	err := prepDirectory(filepath.Dir(c.ConfigPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"location": c.ConfigPath,
	}).Debug("Writing Appconfig")
	err = ioutil.WriteFile(c.ConfigPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	log.Debug("Write successful")
	return nil
}

// Read app configuration from a specified location on disk.
func (c *AppConfig) Read() error {
	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Starting Config Read")

	source, err := ioutil.ReadFile(c.ConfigPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(source, c)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Finished config read")
	return nil
}

// Config goes through a set of prompts to receive user input and generate an AppConfig struct.
func (c *AppConfig) Config() error {
	if c.configExists() {
		fmt.Printf("Editing existing ddev project at %s\n\n", c.AppRoot)
	} else {
		fmt.Printf("Creating a new ddev project config in the current directory (%s)\n", c.AppRoot)
		fmt.Printf("Once completed, you configuration will be written to %s\n\n\n", c.ConfigPath)
	}

	// Log what the starting config is, for debugging purposes.
	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Configuring application")

	var namePrompt string
	if c.Name == "" {
		namePrompt = "Name: "
	} else {
		namePrompt = fmt.Sprintf("Name (%s): ", c.Name)
	}
	// Define an application name.
	fmt.Print(namePrompt)
	c.Name = getInput(c.Name)

	c.docrootPrompt()

	err := c.appTypePrompt()
	if err != nil {
		return err
	}

	if !system.FileExists(c.dockerComposeYAMLPath()) {
		c.WriteDockerComposeConfig()
	}

	// Log the resulting config, for debugging purposes.
	log.WithFields(log.Fields{
		"Config": pretty.Prettify(c),
	}).Debug("Configuration completed")

	return nil
}

// dockerComposeYAMLPath returns the absolute path to where the docker-compose.yaml should exist for this app configuration.
func (c *AppConfig) dockerComposeYAMLPath() string {
	return path.Join(c.AppRoot, ".ddev", "docker-compose.yaml")
}

// URL returns the URL to the app controlled by this config.
func (c *AppConfig) URL() string {
	return c.Name + "." + DDevTLD
}

// WriteDockerComposeConfig writes a docker-compose.yaml to the app configuration directory.
func (c *AppConfig) WriteDockerComposeConfig() error {
	log.WithFields(log.Fields{
		"Location": c.dockerComposeYAMLPath(),
	}).Debug("Writing docker-compose.yaml")

	f, err := os.Create(c.dockerComposeYAMLPath())
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rendered, err := c.RenderComposeYAML()
	if err != nil {
		return err
	}
	_, err = f.WriteString(rendered)

	return err
}

// RenderComposeYAML renders the contents of docker-compose.yaml.
func (c *AppConfig) RenderComposeYAML() (string, error) {
	var doc bytes.Buffer
	var err error
	templ := template.New("compose template")
	templ, err = templ.Parse(DDevComposeTemplate)
	if err != nil {
		return "", err
	}
	templateVars := map[string]string{
		"name":    c.Name,
		"tld":     DDevTLD,
		"docroot": filepath.Join("../", c.Docroot),
		// @TODO: this absolutely needs to come from outside the appconfig package.
		"app_url": c.URL(),
	}

	templ.Execute(&doc, templateVars)
	return doc.String(), nil
}

func (c *AppConfig) docrootPrompt() error {
	// Determine the document root.
	var basePrompt = "Docroot Location"
	var docrootPrompt string
	if c.Docroot == "" {
		docrootPrompt = fmt.Sprintf("%s: ", basePrompt)
	} else {
		docrootPrompt = fmt.Sprintf("%s (%s): ", basePrompt, c.Docroot)
	}
	fmt.Print(docrootPrompt)
	c.Docroot = getInput(c.Docroot)

	// Ensure the docroot exists. If it doesn't, prompt the user to verify they entered it correctly.
	fullPath := filepath.Join(c.AppRoot, c.Docroot)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("No directory could be found at %s. Are you sure this is where your docroot is location? (y/N): ", fullPath)
		answer := strings.ToLower(getInput("y"))
		if answer != "y" && answer != "yes" {
			return c.docrootPrompt()
		}
	}
	return nil
}

func (c *AppConfig) configExists() bool {
	if _, err := os.Stat(c.ConfigPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// appTypePrompt handles the AppType workflow.
func (c *AppConfig) appTypePrompt() error {
	var appType string
	basePrompt := fmt.Sprintf("Application Type [%s]", strings.Join(allowedAppTypes, ", "))

	// First, see if we can auto detect what kind of site it is so we can set a sane default.
	absDocroot := filepath.Join(c.AppRoot, c.Docroot)
	log.WithFields(log.Fields{
		"Location": absDocroot,
	}).Debug("Attempting to auto-determine application type")

	appType, err := determineAppType(absDocroot)
	if err == nil {
		// If we found an application type just set it and inform the user.
		fmt.Printf("Found a %s codebase at %s\n", appType, c.Docroot)
		c.AppType = appType
		return nil
	}

	// If we weren't able to auto-detect the app type, then prompt the user.
	var typePrompt string
	if c.AppType == "" {
		typePrompt = fmt.Sprintf("%s: ", basePrompt)
	} else {
		typePrompt = fmt.Sprintf("%s (%s): ", basePrompt, c.AppType)
	}

	for isAllowedAppType(appType) != true {
		fmt.Printf(typePrompt)
		appType = strings.ToLower(getInput(c.AppType))

		if isAllowedAppType(appType) != true {
			fmt.Printf("%s is not a valid application type. Allowed application types are: %s\n", appType, strings.Join(allowedAppTypes, ", "))
		} else {
			c.AppType = appType
		}
	}
	return nil
}

// getInput reads input from an input buffer and returns the result as a string.
func getInput(defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Could not read input: %s\n", err)
	}

	// If the value from the input buffer is blank, then use the default instead.
	value := strings.TrimSpace(input)
	if value == "" {
		value = defaultValue
	}

	return value
}

// isAllowedAppType determines if a given string exists in the allowedAppTypes slice.
func isAllowedAppType(appType string) bool {
	for _, t := range allowedAppTypes {
		if appType == t {
			return true
		}
	}
	return false
}

// prepDirectory creates a .ddev directory in the current working
func prepDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {

		log.WithFields(log.Fields{
			"directory": dir,
		}).Debug("Config Directory does not exist, attempting to create.")

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"directory": dir,
		}).Debug("Directory creation successful")
	}

	return nil
}

// DetermineAppType uses some predetermined file checks to determine if a local app
// is of any of the known types
func determineAppType(basePath string) (string, error) {
	defaultLocations := map[string]string{
		"scripts/drupal.sh":      "drupal7",
		"core/scripts/drupal.sh": "drupal8",
		"wp": "wordpress",
	}

	for k, v := range defaultLocations {
		fp := path.Join(basePath, k)

		log.WithFields(log.Fields{
			"file": fp,
		}).Debug("Looking for app fingerprint.")
		if system.FileExists(fp) {
			log.WithFields(log.Fields{
				"file": fp,
				"app":  v,
			}).Debug("Found app fingerprint.")

			return v, nil
		}
	}

	return "", fmt.Errorf("Couldn't determine app's type!")
}
