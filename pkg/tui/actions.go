package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/config/types"
)

// action is what a view asks of the app in reply to a key or message. It is
// applied at once, so the reply shows in the same update.
type action any

type (
	openDetailAction struct{ project ProjectInfo }
	// backAction leaves the detail for the dashboard.
	backAction struct{}
	// leaveStreamAction leaves the logs for the view they came from.
	leaveStreamAction struct{}
	// operationAction hands the terminal to a ddev subcommand in dir.
	operationAction struct {
		title, dir string
		args       []string
		// done replaces "<title>: done" in the status line when it succeeds.
		done string
		// shell is set for an interactive shell, which exits with the status
		// of the user's last command, so a non-zero exit is not a failure.
		shell bool
		// selectCurrent selects the working directory's project once it
		// is listed, for `ddev config`.
		selectCurrent bool
	}
	logsAction    struct{ name, appRoot string }
	confirmAction pendingConfirm
	refreshAction struct{}
	statusAction  struct {
		text  string
		isErr bool
		cmd   tea.Cmd
	}
)

func operation(title, dir string, args ...string) operationAction {
	return operationAction{title: title, dir: dir, args: args}
}

// browse opens the site, Mailpit or XHGui of a project, depending on which
// key was pressed. It runs as an operation, since ddev starts or repairs the
// project first when it needs to, and xhgui may start its service.
func browse(k KeyMap, msg tea.KeyPressMsg, name, appRoot string) operationAction {
	switch {
	case key.Matches(msg, k.Mailpit):
		return operation("Opening Mailpit of "+name, appRoot, "mailpit")
	case key.Matches(msg, k.XHGui):
		return operation("Opening XHGui of "+name, appRoot, "xhgui")
	}
	return operation("Launching "+name, appRoot, "launch")
}

// usesXHGui reports whether XHGui can open for a project in xhprofMode.
// `ddev xhgui` starts a stopped project before it checks the mode, so the
// key is off rather than left to fail after the start.
func usesXHGui(xhprofMode string) bool {
	return xhprofMode == types.XHProfModeXHGui
}

func (m AppModel) apply(act action) (AppModel, tea.Cmd) {
	switch a := act.(type) {
	case openDetailAction:
		return m.openDetail(a.project)
	case backAction:
		m.view = viewDashboard
		m.detail = detailView{}
	case leaveStreamAction:
		m, drain := m.stopStream()
		m, reload := m.reload()
		return m, tea.Batch(drain, reload)
	case operationAction:
		m.help.ShowAll = false
		return m, execDdev(a)
	case logsAction:
		m = m.newStream("Logs: " + a.name)
		return m, startStream(m.stream.id, a.appRoot, "logs", "-f")
	case confirmAction:
		c := pendingConfirm(a)
		m.confirm = &c
	case refreshAction:
		return m.refresh()
	case statusAction:
		if a.isErr {
			return m.setError(a.text), a.cmd
		}
		m, expire := m.setStatus(a.text)
		return m, tea.Batch(expire, a.cmd)
	}
	return m, nil
}

type confirmKind int

const (
	confirmStartAll confirmKind = iota
	confirmStopAll
	confirmPoweroff
	confirmDeleteProject
	confirmRestore
	confirmUpdate
)

// pendingConfirm is an action waiting for y (or O for a delete).
type pendingConfirm struct {
	kind   confirmKind
	target string
	// dir is the project root the restore and update run in.
	dir string
	// args are the add-ons to update, so that no more than the prompt lists
	// are.
	args   []string
	prompt string
}

// handleConfirm runs the pending action on y, or a delete without a snapshot
// on O, and cancels it on any other key.
func (m AppModel) handleConfirm(msg tea.KeyPressMsg) (AppModel, tea.Cmd) {
	c := *m.confirm
	m.confirm = nil
	omitSnapshot := c.kind == confirmDeleteProject && key.Matches(msg, m.keys.OmitSnap)
	if !omitSnapshot && !key.Matches(msg, m.keys.Confirm) {
		return m, nil
	}
	switch c.kind {
	case confirmStartAll:
		return m.apply(operation("Starting all projects", "", "start", "--all"))
	case confirmStopAll:
		return m.apply(operation("Stopping all projects", "", "stop", "--all"))
	case confirmPoweroff:
		return m.apply(operation("Powering off DDEV", "", "poweroff"))
	case confirmRestore:
		return m.apply(operation("Restoring the latest snapshot of "+c.target, c.dir, "snapshot", "restore", "--latest"))
	case confirmUpdate:
		op := operation("Updating the add-ons of "+c.target, c.dir, append([]string{"add-on", "update", "--yes"}, c.args...)...)
		op.done = "Updated the add-ons of " + c.target + ", restart with " + m.keys.Restart.Help().Key + " to use them"
		return m.apply(op)
	}
	// A deleted project has no detail view to return to, and a load of it
	// still running must not apply.
	m.view, m.detail = viewDashboard, detailView{}
	if omitSnapshot {
		return m.apply(operation("Deleting "+c.target+" without a snapshot", "", "delete", "-y", "--omit-snapshot", c.target))
	}
	return m.apply(operation("Deleting "+c.target, "", "delete", "-y", c.target))
}

func (m AppModel) confirmHelp() helpKeys {
	omit := m.keys.OmitSnap
	omit.SetEnabled(m.confirm.kind == confirmDeleteProject)
	keys := []key.Binding{m.keys.Confirm, omit, m.keys.Cancel}
	return helpKeys{short: keys, full: [][]key.Binding{keys}}
}
