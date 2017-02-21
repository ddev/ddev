// Package plugins contains the various implementations of the Action interface
// which when implemented can add functionality to the dcfg tool
package plugins

import (
	"github.com/drud/dcfg/tpl"
	"github.com/drud/drud-go/utils/pretty"
)

// Task is the interface that eash plugin must implement
type Task interface {
	String() string
	Run() error
}

// TaskDefaults is a parent type that can be used in new plugins to import these common feilds
type TaskDefaults struct {
	Name    string `yaml:"name"`    // name of the task
	Dest    string `yaml:"dest"`    // what this action will be performed on
	Workdir string `yaml:"workdir"` // where this action will be called from
	Wait    string `yaml:"wait"`    // how long to wait before this action is called
	Repeat  int    `yaml:"repeat"`  // how many times to run this action
	Ignore  bool   `yaml:"ignore"`  // ignore failures or not
}

// String prints the Task
func (t TaskDefaults) String() string {
	return pretty.Prettify(t)
}

// TaskType is used so we can choose which Action implementation to use
type TaskType struct {
	Action string `yaml:"action"` // which action is being called
}

// TypeMap is used to retrieve the correct plugin
var TypeMap = map[string]Task{
	"command":  &Command{},
	"write":    &Write{},
	"replace":  &Replace{},
	"config":   &Config{},
	"template": &tpl.Config{},
}
