package appconfig

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drud/ddev/pkg/version"
	"github.com/drud/drud-go/utils/pretty"
	yaml "gopkg.in/yaml.v2"
)

const CurrentAppVersion = "1"

var allowedAppTypes = []string{"drupal7", "drupal8", "wordpress"}

// AppConfig defines the yaml config file format for ddev applications
type AppConfig struct {
	AppVersion string `yaml:"version"`
	Name       string `yaml:"name"`
	AppType    string `yaml:"type"`
	Docroot    string `yaml:"docroot"`
	WebImage   string `yaml:"webimage"`
	DBImage    string `yaml:"dbimage"`
	FilePath   string
}

// NewAppConfig creates a new AppConfig struct with defaults set. It is preferred to using new() directly.
func NewAppConfig(FilePath string) (*AppConfig, error) {
	// Set defaults.
	c := &AppConfig{}
	c.FilePath = FilePath

	// We're not currently doing anything with AppVersion, so just default it to 1 for now.
	c.AppVersion = "1"

	// These should always default to the latest image/tag names from the Version package.
	c.WebImage = version.WebImg + ":" + version.WebTag
	c.DBImage = version.DBImg + ":" + version.DBTag

	// Load from file if available. This will return an error if the file doesn't exist,
	// and it is up to the caller to determine if that's an issue.
	err := c.Read()
	return c, err
}

// Write the app configuration to a specific location on disk
func (c *AppConfig) Write() error {
	log.WithFields(log.Fields{
		"location": c.FilePath,
	}).Debug("Writing Appconfig")

	cfgbytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(c.FilePath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Read app configuration from a specified location on disk.
func (c *AppConfig) Read() error {
	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Starting Config Read")

	source, err := ioutil.ReadFile(c.FilePath)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(source, c)
	if err != nil {
		panic(err)
	}

	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Finished Config Read")
	return nil
}

// Config goes through a set of prompts to receive user input and generate an AppConfig struct.
func (c *AppConfig) Config() error {
	// Log what the starting config is, for debugging purposes.
	log.WithFields(log.Fields{
		"Existing config": pretty.Prettify(c),
	}).Debug("Configuring application")

	// Default Version
	if c.AppVersion == "" {
		c.AppVersion = "1"
	}

	// Define an application name.
	fmt.Print("Name: ")
	c.Name = getInput()

	// Determine the application type.
	var appType string
	for isAllowedAppType(appType) != true {
		fmt.Printf("Application Type [%s]: ", strings.Join(allowedAppTypes, ", "))
		appType = strings.ToLower(getInput())

		if isAllowedAppType(appType) != true {
			fmt.Printf("%s is not a valid application type. Allowed application types are: %s\n", appType, strings.Join(allowedAppTypes, ", "))
		} else {
			c.AppType = appType
		}
	}

	// Determine the document root.
	// @TODO: Should check to ensure the docroot exists here.
	fmt.Print("Docroot Location: ")
	c.Docroot = getInput()

	// Log the resulting config, for debugging purposes.
	log.WithFields(log.Fields{
		"Config": pretty.Prettify(c),
	}).Debug("Configuration completed")

	return nil
}

// getInput reads input from an input buffer and returns the result as a string.
func getInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Could not read input: %s\n", err)
	}

	return strings.TrimSpace(input)
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
