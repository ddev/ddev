package plugins

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/drud/drud-go/utils/pretty"
)

// Config implements the Config Action
type Config struct {
	TaskDefaults
	Delim string            `yaml:"delim"` // what separates the key and value for this config?
	Items map[string]string // key=>value pairs that should be in the config file
}

func (c Config) String() string {
	return pretty.Prettify(c)
}

// Run executes the command task
func (c *Config) Run() error {
	if c.Wait != "" {
		lengthOfWait, _ := time.ParseDuration(c.Wait)
		time.Sleep(lengthOfWait)
	}

	mode := os.FileMode(0774)
	var confFile []byte
	var err error

	fInfo, err := os.Stat(c.Dest)
	if !os.IsNotExist(err) {
		mode = fInfo.Mode()
		confFile, err = ioutil.ReadFile(c.Dest)
		if err != nil {
			return err
		}
	}

	for k, v := range c.Items {
		entry := strings.Join([]string{k, v + "\n"}, c.Delim)
		if !strings.Contains(string(confFile), entry) && !strings.Contains(string(confFile), k+c.Delim) {
			confFile = append(confFile, entry...)
		} else if strings.Contains(string(confFile), k+c.Delim) {
			re := regexp.MustCompile(fmt.Sprintf("(%s%s)(.*)", k, c.Delim))
			newContent := re.ReplaceAllString(string(confFile), "${1}"+v)
			confFile = []byte(newContent)
		}
	}

	err = ioutil.WriteFile(c.Dest, confFile, mode)
	if err != nil {
		if !c.Ignore {
			return err
		}
	}

	return nil
}
