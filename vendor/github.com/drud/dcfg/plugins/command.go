package plugins

import (
	"fmt"
	"strings"
	"time"

	"github.com/drud/drud-go/utils/pretty"
	"github.com/drud/drud-go/utils/system"
)

// Command implements the Command Action
type Command struct {
	TaskDefaults
	Cmd string `yaml:"cmd"`
}

func (c Command) String() string {
	return pretty.Prettify(c)
}

// Run executes the command task
func (c *Command) Run() error {

	for i := c.Repeat; i >= 0; i-- {

		if c.Wait != "" {
			lengthOfWait, _ := time.ParseDuration(c.Wait)
			time.Sleep(lengthOfWait)
		}

		taskPayload := c.Cmd
		if taskPayload == "" {
			return fmt.Errorf("No cmd specified")
		}

		parts := strings.Split(taskPayload, " ")

		err := system.RunCommandPipe(parts[0], parts[1:])
		if err != nil {
			if !c.Ignore {
				return err
			}
		}

	}
	return nil
}
