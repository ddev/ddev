package tui

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ddev/ddev/pkg/ddevapp"
)

// loadProjects fetches the project list in the background.
func loadProjects() tea.Msg {
	apps, err := ddevapp.GetProjects(false)
	if err != nil {
		return projectsLoadedMsg{err: err}
	}

	projects := make([]ProjectInfo, 0, len(apps))
	for _, app := range apps {
		projects = append(projects, extractProjectInfo(app))
	}
	return projectsLoadedMsg{projects: projects}
}

// ddevExecCommand returns a tea.ExecCommand that runs a ddev subcommand
// as a subprocess, suspending the TUI.
func ddevExecCommand(args ...string) tea.Cmd {
	return ddevExecCommandInDir("", args...)
}

// ddevExecCommandInDir runs a ddev subcommand in the given directory.
// If dir is empty, the current directory is used.
func ddevExecCommandInDir(dir string, args ...string) tea.Cmd {
	ddevBin, err := os.Executable()
	if err != nil {
		return func() tea.Msg {
			return operationFinishedMsg{err: err}
		}
	}

	c := exec.Command(ddevBin, args...)
	c.Env = append(os.Environ(), "DDEV_NO_TUI=true")
	if dir != "" {
		c.Dir = dir
	}

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return operationFinishedMsg{err: err}
	})
}
