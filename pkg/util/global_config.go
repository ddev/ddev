package util

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DdevGlobalConfigName is the name of the global config file.
const DdevGlobalConfigName = "global_config.yaml"

var (
	// DdevGlobalConfig is the currently active global configuration struct
	DdevGlobalConfig GlobalConfig
)

// GlobalConfig is the struct defining ddev's global config
type GlobalConfig struct {
	OmitContainers            []string               `yaml:"omit_containers"`
	LastRunVersion string `yaml:"last_run_version"`
	InstrumentationOptIn bool `yaml:"instrumentation_opt_in"`
}

// GetGlobalConfigPath() gets the path to global config file
func GetGlobalConfigPath() string {
	return filepath.Join(GetGlobalDdevDir(),DdevGlobalConfigName)

}

// ReadGlobalConfig() reads the global config file into DdevGlobalConfig
func ReadGlobalConfig() error {
	globalConfigFile := GetGlobalConfigPath()
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
	return nil
}

// WriteGlobalConfig writes the global config into ~/.ddev.
func WriteGlobalConfig(config GlobalConfig) error {
	cfgbytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(GetGlobalConfigPath(), cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
