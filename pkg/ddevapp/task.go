package ddevapp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
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
	service string
	exec    string
	app     *DdevApp
}

// ExecHostTask is the struct that defines "exec-host" tasks for hooks,
// commands that get run on the host.
type ExecHostTask struct {
	exec string
	app  *DdevApp
}

type ComposerTask struct {
	exec string
	app  *DdevApp
}

// Execute() executes an ExecTask
func (c ExecTask) Execute() error {
	_, _, err := c.app.Exec(&ExecOpts{
		Service:   c.service,
		Cmd:       c.exec,
		Tty:       isatty.IsTerminal(os.Stdin.Fd()),
		NoCapture: true,
	})

	return err
}

// GetDescription returns a human-readable description of the task
func (c ExecTask) GetDescription() string {
	return fmt.Sprintf("Exec command '%s' in container/service '%s'", c.exec, c.service)
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

	bashPath, err := util.FindBashPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(bashPath, "-c", c.exec)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	_ = os.Chdir(cwd)

	return err
}

// Execute (ComposerTask) runs a composer command in the web container
// and returns stdout, stderr, err
func (c ComposerTask) Execute() error {
	components := strings.Split(c.exec, " ")
	_, _, err := c.app.Composer(components[0:])

	return err
}

// GetDescription returns a human-readable description of the task
func (c ComposerTask) GetDescription() string {
	return fmt.Sprintf("Composer command '%s' in web container", c.exec)
}

// NewTask() is the factory method to create whatever kind of task
// we need using the yaml description of the task.
// Returns a task (of various types) or nil
func NewTask(app *DdevApp, ytask YAMLTask) Task {
	if e, ok := ytask["exec-host"]; ok {
		t := ExecHostTask{app: app, exec: e.(string)}
		return t
	} else if e, ok = ytask["exec"]; ok {
		t := ExecTask{app: app, exec: e.(string)}
		if t.service, ok = ytask["service"].(string); !ok {
			t.service = nodeps.WebContainer
		}
		return t
	} else if e, ok = ytask["composer"]; ok {
		t := ComposerTask{app: app, exec: e.(string)}
		return t
	}
	return nil
}
