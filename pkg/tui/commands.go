package tui

import (
	"fmt"
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

// loadDetailCmd fetches full project detail in the background.
func loadDetailCmd(appRoot string) tea.Cmd {
	return func() tea.Msg {
		app, err := ddevapp.NewApp(appRoot, true)
		if err != nil {
			return projectDetailLoadedMsg{err: err}
		}

		desc, err := app.Describe(false)
		if err != nil {
			return projectDetailLoadedMsg{err: err}
		}

		detail := ProjectDetail{
			Name:    app.Name,
			AppRoot: appRoot,
		}

		detail.Status, _ = desc["status"].(string)
		detail.Type, _ = desc["type"].(string)
		detail.PHPVersion, _ = desc["php_version"].(string)
		detail.WebserverType, _ = desc["webserver_type"].(string)
		detail.NodeJSVersion, _ = desc["nodejs_version"].(string)
		detail.Docroot, _ = desc["docroot"].(string)
		detail.DatabaseType, _ = desc["database_type"].(string)
		detail.DatabaseVersion, _ = desc["database_version"].(string)
		detail.XdebugEnabled, _ = desc["xdebug_enabled"].(bool)
		detail.PerformanceMode, _ = desc["performance_mode"].(string)

		if urls, ok := desc["urls"].([]string); ok {
			detail.URLs = urls
		}

		detail.MailpitURL, _ = desc["mailpit_https_url"].(string)
		if detail.MailpitURL == "" {
			detail.MailpitURL, _ = desc["mailpit_url"].(string)
		}

		if dbInfo, ok := desc["dbinfo"].(map[string]interface{}); ok {
			if port, ok := dbInfo["published_port"].(int); ok {
				detail.DBPublishedPort = fmt.Sprintf("127.0.0.1:%d", port)
			}
		}

		detail.Addons = ddevapp.GetInstalledAddonNames(app)

		if services, ok := desc["services"].(map[string]map[string]interface{}); ok {
			for name, svc := range services {
				status, _ := svc["status"].(string)
				detail.Services = append(detail.Services, ServiceInfo{Name: name, Status: status})
			}
		}

		return projectDetailLoadedMsg{detail: detail}
	}
}

// loadLogsCmd fetches container logs in the background.
func loadLogsCmd(appRoot string, service string) tea.Cmd {
	return func() tea.Msg {
		app, err := ddevapp.NewApp(appRoot, true)
		if err != nil {
			return logsLoadedMsg{err: err, service: service}
		}

		logs, err := app.CaptureLogs(service, false, "100")
		return logsLoadedMsg{logs: logs, service: service, err: err}
	}
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

// ddevExecCommandDetail runs a ddev subcommand and returns operationDetailFinishedMsg.
func ddevExecCommandDetail(dir string, args ...string) tea.Cmd {
	ddevBin, err := os.Executable()
	if err != nil {
		return func() tea.Msg {
			return operationDetailFinishedMsg{err: err}
		}
	}

	c := exec.Command(ddevBin, args...)
	c.Env = append(os.Environ(), "DDEV_NO_TUI=true")
	if dir != "" {
		c.Dir = dir
	}

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return operationDetailFinishedMsg{err: err}
	})
}
