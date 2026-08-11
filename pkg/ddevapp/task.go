package ddevapp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
)

// YAMLTask defines tasks like Exec to be run in hooks
type YAMLTask map[string]any

// Task is the interface defining methods we'll use in various tasks
type Task interface {
	Execute() error
	GetDescription() string
}

// ExecTask is the struct that defines "exec" tasks for hooks, commands
// to be run in containers.
type ExecTask struct {
	service string   // Name of service, defaults to web
	user    string   // User to run the command as
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

// Execute executes an ExecTask. Output is teed to a buffer so a failure's
// error can carry it, the same way ExecOnHostOrService's does for providers;
// the command still streams live exactly as before.
func (c ExecTask) Execute() error {
	var captured bytes.Buffer
	opts := &ExecOpts{
		Service:   c.service,
		User:      c.user,
		Cmd:       c.exec,
		RawCmd:    c.execRaw,
		Tty:       isatty.IsTerminal(os.Stdin.Fd()),
		NoCapture: true,
		Stdout:    io.MultiWriter(os.Stdout, &captured),
		Stderr:    io.MultiWriter(os.Stderr, &captured),
	}
	_, _, err := c.app.Exec(opts)
	if err != nil {
		return errorWithOutput(err, captured.String())
	}
	return nil
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

// Execute (HostTask) executes a command on the host, keeping stdin and the
// terminal connected, and attaches captured output to a failure the same
// way ExecOnHostOrService's host branch does for providers.
func (c ExecHostTask) Execute() error {
	return runHostCommandWithCapturedOutput(c.app, c.exec)
}

// Execute (ComposerTask) runs a Composer command in the web container
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
	if value, ok := ytask["exec-host"]; ok {
		if v, ok := value.(string); ok {
			t := ExecHostTask{app: app, exec: v}
			return t
		}
		util.Warning("Invalid exec-host value, not executing it: %v", value)
	} else if value, ok = ytask["composer"]; ok {
		// Handle the old-style `composer: install`
		if v, ok := value.(string); ok {
			t := ComposerTask{app: app, execRaw: strings.Split(v, " ")}
			return t
		}

		// Handle the new-style `composer: [install]`
		if v, ok := ytask["exec_raw"]; ok {
			raw, err := util.InterfaceSliceToStringSlice(v)
			if err != nil {
				util.Warning("Invalid composer/exec_raw value, not executing it: %v", err)
				return nil
			}

			t := ComposerTask{app: app, execRaw: raw}
			return t
		}
		util.Warning("Invalid Composer value, not executing it: %v", value)
	} else if value, ok = ytask["exec"]; ok {
		if v, ok := value.(string); ok {
			t := ExecTask{app: app, exec: v}
			if t.service, ok = ytask["service"].(string); !ok {
				t.service = nodeps.WebContainer
			}
			// Handle user as either string or integer
			if userVal, ok := ytask["user"].(string); ok {
				t.user = userVal
			} else if userVal, ok := ytask["user"].(int); ok {
				t.user = fmt.Sprintf("%d", userVal)
			} else {
				t.user = ""
			}
			return t
		}

		if v, ok := ytask["exec_raw"]; ok {
			raw, err := util.InterfaceSliceToStringSlice(v)
			if err != nil {
				util.Warning("Invalid exec/exec_raw value, not executing it: %v", err)
				return nil
			}

			t := ExecTask{app: app, execRaw: raw}
			if t.service, ok = ytask["service"].(string); !ok {
				t.service = nodeps.WebContainer
			}
			// Handle user as either string or integer
			if userVal, ok := ytask["user"].(string); ok {
				t.user = userVal
			} else if userVal, ok := ytask["user"].(int); ok {
				t.user = fmt.Sprintf("%d", userVal)
			} else {
				t.user = ""
			}
			return t
		}
		util.Warning("Invalid exec/exec_raw value, not executing it: %v", value)
	}
	return nil
}
