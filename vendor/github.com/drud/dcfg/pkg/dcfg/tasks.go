// Package dcfg groups.go - The TaskSet and TaskSetList types represent the blocks of commands
// contained in your drud.yaml(e.g. the install and uninstall examples from the readne).
package dcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/dcfg/plugins"
)

// TaskSet models the outermost list item in the drud.yaml file
type TaskSet struct {
	Name    string             `yaml:"name",json:"name"`
	Env     map[string]string  `yaml:"env",json:"env"`
	User    string             `yaml:"user",json:"user"`
	Workdir string             `yaml:"workdir",json:"workdir"`
	Tasks   []*json.RawMessage `json:"tasks",yaml:"tasks"`
}

// Run will move to any directory specified by Workdir, switch to the User if specified then
// loop through the Tasks replacing templating vars with values from the Env map. It then determines
// which implementation of the plugins.Action interface to use for each task. After that it
// calls the Run method on the individual tasks.
func (g *TaskSet) Run() error {
	baseDir, _ := os.Getwd()
	var workDir string
	if g.Workdir != "" {
		workDir, _ = filepath.Abs(g.Workdir)
	}
	maybeChdir(workDir)

	// uppercase env values that start with $ are assumed to be referencing
	// environment variables in the host
	for k, v := range g.Env {
		if strings.HasPrefix(v, "$") && v == strings.ToUpper(v) {
			g.Env[k] = os.Getenv(v[1:])
		}
	}

	fmt.Println("Running group:", g.Name)

	for _, t := range g.Tasks {
		taskString := string([]byte(*t))

		// Determine if there are tempalte strings in the task and replace then with values
		// from the TaskSet.Env map
		if HasVars(taskString) {
			var doc bytes.Buffer
			templ := template.New("cmd template")
			templ, _ = templ.Parse(taskString)
			templ.Execute(&doc, g.Env)
			taskString = doc.String()
		}

		// first Unmarshal into the TaskType so we can use the Action field to determine which plugin
		// to implement for the given task
		var cmdType plugins.TaskType
		err := json.Unmarshal([]byte(taskString), &cmdType)
		if err != nil {
			fmt.Println(err)
		}

		// each plugin is registered in the plugins.TypeMap which will use the action string
		// as the index in order to get the plugin it is mapped to
		action := plugins.TypeMap[cmdType.Action]
		err = json.Unmarshal([]byte(taskString), &action)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(action)
		err = action.Run()
		if err != nil {
			log.Fatal(err)
		}

	}

	os.Chdir(baseDir)

	return nil
}

// TaskSetList represents a lsit of TaskSets
type TaskSetList []TaskSet
