package platform

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/drud/drud-go/utils/system"
	yaml "gopkg.in/yaml.v2"
)

// Config represents the data that is or can be stored in $HOME/.drud
type Config struct {
	APIVersion      string `yaml:"apiversion"`
	ActiveApp       string `yaml:"activeapp"`
	ActiveDeploy    string `yaml:"activedeploy"`
	Client          string `yaml:"client"`
	DrudHost        string `yaml:"drudhost"`
	GithubAuthToken string `yaml:"githubauthtoken"`
	GithubAuthOrg   string `yaml:"githubauthorg"`
	Protocol        string `yaml:"protocol"`
	VaultAddr       string `yaml:"vaultaddr"`
	VaultAuthToken  string `yaml:"vaultauthtoken"`
	Workspace       string `yaml:"workspace"`
}

func parseConfigFlag() string {
	var value string

	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--config=") {
			value = strings.TrimPrefix(arg, "--config=")
		} else if arg == "--config" {
			value = os.Args[i+1]
		}
	}
	if value == "" {
		home, _ := system.GetHomeDir()
		value = fmt.Sprintf("%v/drud.yaml", home)
	}

	if _, err := os.Stat(value); os.IsNotExist(err) {
		var cFile, err = os.Create(value)
		if err != nil {
			log.Fatal(err)
		}
		cFile.Close()
	}
	return value
}

// GetConfig Loads a config structure from yaml and environment.
func GetConfig() (cfg *Config, err error) {
	cfgFile := parseConfigFlag()

	source, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	c := &Config{}
	err = yaml.Unmarshal(source, c)
	if err != nil {
		panic(err)
	}

	if c.APIVersion == "" && os.Getenv("DRUD_APIVERSION") != "" {
		c.APIVersion = os.Getenv("DRUD_APIVERSION")
	}
	if c.ActiveApp == "" && os.Getenv("DRUD_ACTIVEAPP") != "" {
		c.ActiveApp = os.Getenv("DRUD_ACTIVEAPP")
	}
	if c.ActiveDeploy == "" && os.Getenv("DRUD_ACTIVEDEPLOY") != "" {
		c.ActiveDeploy = os.Getenv("DRUD_ACTIVEDEPLOY")
	}
	if c.Client == "" && os.Getenv("DRUD_CLIENT") != "" {
		c.Client = os.Getenv("DRUD_CLIENT")
	}
	if c.DrudHost == "" && os.Getenv("DRUD_DRUDHOST") != "" {
		c.DrudHost = os.Getenv("DRUD_DRUDHOST")
	}
	if c.GithubAuthToken == "" && os.Getenv("DRUD_GITHUBAUTHTOKEN") != "" {
		c.GithubAuthToken = os.Getenv("DRUD_GITHUBAUTHTOKEN")
	}
	if c.GithubAuthOrg == "" && os.Getenv("DRUD_GITHUBAUTHORG") != "" {
		c.GithubAuthOrg = os.Getenv("DRUD_GITHUBAUTHORG")
	}
	if c.Protocol == "" && os.Getenv("DRUD_PROTOCOL") != "" {
		c.Protocol = os.Getenv("DRUD_PROTOCOL")
	}
	if c.VaultAddr == "" && os.Getenv("DRUD_VAULTADDR") != "" {
		c.VaultAddr = os.Getenv("DRUD_VAULTADDR")
	}
	if c.VaultAuthToken == "" && os.Getenv("DRUD_VAULTAUTHTOKEN") != "" {
		c.VaultAuthToken = os.Getenv("DRUD_VAULTAUTHTOKEN")
	}
	if c.Workspace == "" && os.Getenv("DRUD_WORKSPACE") != "" {
		c.Workspace = os.Getenv("DRUD_WORKSPACE")
	}

	return c, nil
}

// WriteConfig writes each config value to the BoltDB and updates the
// global cfg as well.
func (cfg *Config) WriteConfig(f string) (err error) {
	cfgbytes, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(f, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}
