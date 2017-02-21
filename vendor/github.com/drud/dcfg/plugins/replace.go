package plugins

import (
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/drud/drud-go/utils/pretty"
)

// Replace implements the Replace Action
type Replace struct {
	TaskDefaults
	Find    string `yaml:"find"`    // the needle or a regular expression
	Replace string `yaml:"replace"` // what to replace the needle with
}

func (r Replace) String() string {
	return pretty.Prettify(r)
}

// Run executes the command task
func (r *Replace) Run() error {
	mode := os.FileMode(0774)

	fInfo, err := os.Stat(r.Dest)
	if !os.IsNotExist(err) {
		mode = fInfo.Mode()
	}

	if r.Wait != "" {
		lengthOfWait, _ := time.ParseDuration(r.Wait)
		time.Sleep(lengthOfWait)
	}

	fbytes, err := ioutil.ReadFile(r.Dest)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(r.Find)
	newContent := re.ReplaceAllString(string(fbytes), r.Replace)

	err = ioutil.WriteFile(r.Dest, []byte(newContent), mode)
	if err != nil {
		if !r.Ignore {
			return err
		}
	}

	return nil
}
