package globalconfig

import (
	"fmt"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DdevGlobalConfigName is the name of the global config file.
const DdevGlobalConfigName = "global_config.yaml"

var (
	// DdevGlobalConfig is the currently active global configuration struct
	DdevGlobalConfig GlobalConfig
)

// GlobalConfig is the struct defining ddev's global config
type GlobalConfig struct {
	APIVersion           string   `yaml:"APIVersion"`
	OmitContainers       []string `yaml:"omit_containers"`
	InstrumentationOptIn bool     `yaml:"instrumentation_opt_in"`
	LastUsedVersion      string   `yaml:"last_used_version"`
}

// GetGlobalConfigPath() gets the path to global config file
func GetGlobalConfigPath() string {
	return filepath.Join(GetGlobalDdevDir(), DdevGlobalConfigName)

}

// ValidateGlobalConfig validates global config
func ValidateGlobalConfig() error {
	if !IsValidOmitContainers(DdevGlobalConfig.OmitContainers) {
		return fmt.Errorf("Invalid omit_containers: %s, must contain only %s", strings.Join(DdevGlobalConfig.OmitContainers, ","), strings.Join(GetValidOmitContainers(), ",")).(InvalidOmitContainers)
	}

	return nil
}

// ReadGlobalConfig() reads the global config file into DdevGlobalConfig
func ReadGlobalConfig() error {
	globalConfigFile := GetGlobalConfigPath()
	// This is added just so we can see it in global; not checked.
	DdevGlobalConfig.APIVersion = version.DdevVersion

	// Can't use fileutil.FileExists() here because of import cycle.
	if _, err := os.Stat(globalConfigFile); err != nil {
		if os.IsNotExist(err) {

			err := WriteGlobalConfig(DdevGlobalConfig)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	source, err := ioutil.ReadFile(globalConfigFile)
	if err != nil {
		return fmt.Errorf("Unable to read ddev global config file %s: %v", source, err)
	}

	// ReadConfig config values from file.
	err = yaml.Unmarshal(source, &DdevGlobalConfig)
	if err != nil {
		return err
	}

	err = ValidateGlobalConfig()
	if err != nil {
		return err
	}
	return nil
}

// WriteGlobalConfig writes the global config into ~/.ddev.
func WriteGlobalConfig(config GlobalConfig) error {

	err := ValidateGlobalConfig()
	if err != nil {
		return err
	}
	cfgbytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Append current image information
	instructions := "\n# You can turn off usage of the dba (phpmyadmin) container and/or \n# ddev-ssh-agent containers with\n# omit_containers[\"dba\", \"ddev-ssh-agent\"]\n\n# and you can opt in or out of sending instrumentation the ddev developers with \n# instrumentation_opt_in: true # or false\n"
	cfgbytes = append(cfgbytes, instructions...)

	err = ioutil.WriteFile(GetGlobalConfigPath(), cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// GetGlobalDdevDir returns ~/.ddev, the global caching directory
func GetGlobalDdevDir() string {
	userHome, err := homedir.Dir()
	if err != nil {
		logrus.Fatal("could not get home directory for current user. is it set?")
	}
	ddevDir := path.Join(userHome, ".ddev")

	// Create the directory if it is not already present.
	if _, err := os.Stat(ddevDir); os.IsNotExist(err) {
		err = os.MkdirAll(ddevDir, 0700)
		if err != nil {
			logrus.Fatalf("Failed to create required directory %s, err: %v", ddevDir, err)
		}
	}
	return ddevDir
}

// IsValidOmitContainers is a helper function to determine if a the OmitContainers array is valid
func IsValidOmitContainers(containerList []string) bool {
	for _, containerName := range containerList {
		if _, ok := ValidOmitContainers[containerName]; !ok {
			return false
		}
	}
	return true
}

// GetValidOmitContainers is a helper function that returns a list of valid containers for OmitContainers.
func GetValidOmitContainers() []string {
	s := make([]string, 0, len(ValidOmitContainers))

	for p := range ValidOmitContainers {
		s = append(s, p)
	}

	return s
}
