package ddevapp

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
)

// YAMLTask defines tasks like Exec to be run in hooks
type YAMLTask map[string]interface{}

// Task is the interface defining methods we'll use in various tasks
type Task interface {
	Execute() error
	GetDescription() string
}

// ExecTask is the struct that defines "exec" tasks for hooks, commands
// to be run in containers.
type ExecTask struct {
	service string   // Name of service, defaults to web
	execRaw []string // Use execRaw if configured instead of exec
	exec    string   // Actual command to be executed.
	app     *DdevApp
}

// ExecHostTask is the struct that defines "exec-host" tasks for hooks,
// commands that get run on the host.
type ExecHostTask struct {
	exec string
	app  *DdevApp
}

// ComposerTask is the struct that defines "composer" tasks for hooks, commands
// to be run in containers.
type ComposerTask struct {
	execRaw []string
	app     *DdevApp
}

// Execute executes an ExecTask
func (c ExecTask) Execute() error {
	opts := &ExecOpts{
		Service:   c.service,
		Cmd:       c.exec,
		RawCmd:    c.execRaw,
		Tty:       isatty.IsTerminal(os.Stdin.Fd()),
		NoCapture: true,
	}
	_, _, err := c.app.Exec(opts)

	return err
}

// GetDescription returns a human-readable description of the task
func (c ExecTask) GetDescription() string {
	s := c.exec
	if c.execRaw != nil {
		s = fmt.Sprintf("%v (raw)", c.execRaw)
	}

	return fmt.Sprintf("Exec command '%s' in container/service '%s'", s, c.service)
}

// GetDescription returns a human-readable description of the task
func (c ExecHostTask) GetDescription() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("Exec command '%s' on the host (%s)", c.exec, hostname)
}

// Execute (HostTask) executes a command in a container, by default the web container,
// and returns stdout, stderr, err
func (c ExecHostTask) Execute() error {
	cwd, _ := os.Getwd()
	err := os.Chdir(c.app.GetAppRoot())
	if err != nil {
		return err
	}

	bashPath := "bash"
	if runtime.GOOS == "windows" {
		bashPath = util.FindBashPath()
	}

	args := []string{
		"-c",
		c.exec,
	}

	err = exec.RunInteractiveCommand(bashPath, args)

	_ = os.Chdir(cwd)

	return err
}

// Execute (ComposerTask) runs a composer command in the web container
// and returns stdout, stderr, err
func (c ComposerTask) Execute() error {
	_, _, err := c.app.Composer(c.execRaw)

	return err
}

// GetDescription returns a human-readable description of the task
func (c ComposerTask) GetDescription() string {
	return fmt.Sprintf("Composer command '%v' in web container", c.execRaw)
}

// NewTask is the factory method to create whatever kind of task
// we need using the yaml description of the task.
// Returns a task (of various types) or nil
func NewTask(app *DdevApp, ytask YAMLTask) Task {
	if e, ok := ytask["exec-host"]; ok {
		if v, ok := e.(string); ok {
			t := ExecHostTask{app: app, exec: v}
			return t
		}
		util.Warning("Invalid exec-host value, not executing it: %v", e)
	} else if e, ok = ytask["composer"]; ok {
		// Handle the old-style `composer: install`
		if v, ok := e.(string); ok {
			t := ComposerTask{app: app, execRaw: strings.Split(v, " ")}
			return t
		}

		// Handle the new-style `composer: [install]`
		if v, ok := ytask["exec_raw"]; ok {
			raw, err := util.InterfaceSliceToStringSlice(v.([]interface{}))
			if err != nil {
				util.Warning("Invalid composer/exec_raw value, not executing it: %v", e)
				return nil
			}

			t := ComposerTask{app: app, execRaw: raw}
			return t
		}
		util.Warning("Invalid composer value, not executing it: %v", e)
	} else if e, ok = ytask["exec"]; ok {
		if v, ok := e.(string); ok {
			t := ExecTask{app: app, exec: v}
			if t.service, ok = ytask["service"].(string); !ok {
				t.service = nodeps.WebContainer
			}
			return t
		}

		if v, ok := ytask["exec_raw"]; ok {
			raw, err := util.InterfaceSliceToStringSlice(v.([]interface{}))
			if err != nil {
				util.Warning("Invalid exec/exec_raw value, not executing it: %v", e)
				return nil
			}

			t := ExecTask{app: app, execRaw: raw}
			if t.service, ok = ytask["service"].(string); !ok {
				t.service = nodeps.WebContainer
			}
			return t
		}
		util.Warning("Invalid exec_raw value, not executing it: %v", e)

	} else if e, ok = ytask["exec"]; ok {
		if v, ok := e.(string); ok {
			t := ExecTask{app: app, exec: v}
			if t.service, ok = ytask["service"].(string); !ok {
				t.service = nodeps.WebContainer
			}
			return t
		}
		util.Warning("Invalid exec value, not executing it: %v", e)
	}
	return nil
}
