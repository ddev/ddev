package tpl

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"

	"os"

	"github.com/drud/drud-go/utils/pretty"
	"github.com/drud/drud-go/utils/stringutil"
)

// Config implements the Template Action
type Config struct {
	App              string `yaml:"app"`
	Core             string `yaml:"core"`
	ConfigPath       string `yaml:"configPath"`
	DocRoot          string `yaml:"docroot"`
	DatabaseName     string `yaml:"databaseName"`
	DatabaseUsername string `yaml:"databaseUsername"`
	DatabasePassword string `yaml:"databasePassword"`
	DatabaseHost     string `yaml:"databaseHost"`
	DatabaseDriver   string `yaml:"databaseDriver"`
	DatabasePort     int    `yaml:"databasePort"`
	DatabasePrefix   string `yaml:"databasePrefix"`
	IgnoreFiles      bool   `yaml:"ignoreFiles"`
	PublicFiles      string `yaml:"publicFiles"`
	PrivateFiles     string `yaml:"privateFiles"`
	ConfigSyncDir    string `yaml:"configSyncDir"`
	SiteURL          string `yaml:"siteURL"`
	CoreDir          string `yaml:"coreDir"`
	ContentDir       string `yaml:"contentDir"`
	UploadDir        string `yaml:"uploadDir"`
}

// Tpl is the interface that each plugin must implement
type Tpl interface {
	WriteConfig(in *Config) error
	PlaceFiles(in *Config, move bool) error
	WebConfig(in *Config) error
}

// TplMap is used to retrieve the correct plugin
var TplMap = map[string]Tpl{
	"drupal":    &DrupalConfig{},
	"wordpress": &WordpressConfig{},
}

// String prints the Task
func (c Config) String() string {
	return pretty.Prettify(c)
}

// Run creates configurations for an application
func (c *Config) Run() error {
	log.Printf("this is a %s app", c.App)

	app := TplMap[c.App]

	err := app.WriteConfig(c)
	if err != nil {
		return err
	}

	if !c.IgnoreFiles {
		if os.Getenv("DEPLOY_NAME") == "local" {
			app.PlaceFiles(c, true)
		} else {
			app.PlaceFiles(c, false)
		}
	}

	if c.DocRoot != "" {
		app.WebConfig(c)
	}

	return nil
}

// PassTheSalt generates a hash salt
func PassTheSalt() string {
	salt := sha256.New()
	random := stringutil.RandomString(20)
	salt.Write([]byte(random))

	return hex.EncodeToString(salt.Sum(nil))
}

// SlashIt ensures you have a preceding or trailing slash on a string
func SlashIt(val string, trailing bool) string {
	if trailing && !strings.HasSuffix(val, "/") {
		return val + "/"
	} else if !trailing && !strings.HasPrefix(val, "/") {
		return "/" + val
	}
	return val
}
