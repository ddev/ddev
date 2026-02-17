package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

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
		if _, err := os.Stat(appRoot); os.IsNotExist(err) {
			return projectDetailLoadedMsg{err: fmt.Errorf("project directory no longer exists: %s", appRoot)}
		}

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

// startLogStreamCmd starts `ddev logs -f` as a background subprocess and
// streams its output line-by-line into the TUI via a channel.
func startLogStreamCmd(appRoot string) tea.Cmd {
	return func() tea.Msg {
		ddevBin, err := os.Executable()
		if err != nil {
			return logStreamEndedMsg{}
		}

		cmd := exec.Command(ddevBin, "logs", "-f")
		cmd.Dir = appRoot
		cmd.Env = append(os.Environ(), "DDEV_NO_TUI=true")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return logStreamEndedMsg{}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return logStreamEndedMsg{}
		}

		if err := cmd.Start(); err != nil {
			return logStreamEndedMsg{}
		}

		ch := make(chan string, 100)
		// Merge stdout and stderr into the channel
		scan := func(r io.Reader) {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				ch <- scanner.Text()
			}
		}
		go scan(stdout)
		go func() {
			scan(stderr)
			// Wait for process to finish, then close channel
			_ = cmd.Wait()
			close(ch)
		}()

		return logStreamStartedMsg{lines: ch, process: cmd.Process}
	}
}

// waitForLogLineCmd waits for the next line from the log stream channel.
func waitForLogLineCmd(ch <-chan string) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logStreamEndedMsg{}
		}
		return logLineMsg{line: line}
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

// ddevConfigCommand runs `ddev config` interactively, capturing stderr
// so the actual error message can be shown in the TUI status bar.
func ddevConfigCommand() tea.Cmd {
	ddevBin, err := os.Executable()
	if err != nil {
		return func() tea.Msg {
			return operationFinishedMsg{err: err}
		}
	}

	var stderrBuf bytes.Buffer
	c := exec.Command(ddevBin, "config")
	c.Env = append(os.Environ(), "DDEV_NO_TUI=true")
	// Tee stderr to both the terminal (so user sees it) and a buffer (for status bar)
	c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil && stderrBuf.Len() > 0 {
			// Extract the last non-empty line from stderr for a meaningful message
			lines := strings.Split(strings.TrimSpace(stderrBuf.String()), "\n")
			lastLine := strings.TrimSpace(lines[len(lines)-1])
			if lastLine != "" {
				return operationFinishedMsg{err: fmt.Errorf("%s", lastLine)}
			}
		}
		return operationFinishedMsg{err: err}
	})
}

// loadRouterStatus fetches the router health status in the background.
func loadRouterStatus() tea.Msg {
	status, _ := ddevapp.GetRouterStatus()
	return routerStatusMsg{status: status}
}

// xdebugToggleCmd runs `ddev xdebug toggle` in the project directory.
func xdebugToggleCmd(appRoot string) tea.Cmd {
	return func() tea.Msg {
		ddevBin, err := os.Executable()
		if err != nil {
			return xdebugToggledMsg{err: err}
		}

		c := exec.Command(ddevBin, "xdebug", "toggle")
		c.Env = append(os.Environ(), "DDEV_NO_TUI=true")
		c.Dir = appRoot

		err = c.Run()
		return xdebugToggledMsg{err: err}
	}
}

// copyToClipboard copies text to the system clipboard using platform-specific tools.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			// Try wayland first, then X11
			if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.Command("wl-copy")
			} else if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else {
				return clipboardMsg{err: fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")}
			}
		default:
			return clipboardMsg{err: fmt.Errorf("clipboard not supported on %s", runtime.GOOS)}
		}

		pipe, err := cmd.StdinPipe()
		if err != nil {
			return clipboardMsg{err: err}
		}

		if err := cmd.Start(); err != nil {
			return clipboardMsg{err: err}
		}

		_, err = pipe.Write([]byte(text))
		_ = pipe.Close()
		if err != nil {
			return clipboardMsg{err: err}
		}

		return clipboardMsg{err: cmd.Wait()}
	}
}
