package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/nodeps"
)

const (
	refreshInterval = 5 * time.Second
	statusLifetime  = 5 * time.Second
	maxStreamLines  = 5000
)

// execPause is how long an operation's last output stays up before the TUI
// redraws over it.
var execPause = time.Second

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func expireStatusCmd(id int) tea.Cmd {
	return tea.Tick(statusLifetime, func(time.Time) tea.Msg { return statusExpiredMsg{id: id} })
}

// Indirections so tests can see what a key runs without running it.
var (
	execDdev          = execDdevCmd
	xdebugStatus      = xdebugCmd
	latestAddons      = latestAddonsCmd
	startStream       = startStreamCmd
	openTargetCmd     = openTarget
	openInPhpStormCmd = openInPhpStorm
	writeClipboard    = clipboard.WriteAll
)

// ddevBinary returns the ddev binary to run subcommands with. Tests point it
// at a built binary, since os.Executable is the test binary there.
var ddevBinary = os.Executable

// baseEnv is the environment from before the TUI described any project.
// Describe sets the project's variables on this process, and some would
// carry over to a subcommand for another project: a PostgreSQL project's
// DDEV_DB_CONTAINER_COMMAND would start a MySQL database with PostgreSQL
// flags. Unset, as in tests, subcommands get the current environment.
var baseEnv []string

// ddevCommand prepares a ddev subcommand in dir (the current directory if
// empty) that cannot start a nested TUI.
func ddevCommand(dir string, args ...string) (*exec.Cmd, error) {
	bin, err := ddevBinary()
	if err != nil {
		return nil, err
	}
	c := exec.Command(bin, args...)
	env := baseEnv
	if env == nil {
		env = os.Environ()
	}
	c.Env = append(slices.Clone(env), "DDEV_NO_TUI=true")
	c.Dir = dir
	return c, nil
}

// lastLine returns the last non-blank line of s as plain text, for a
// message in the status line.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(ansi.Strip(cleanLine(lines[len(lines)-1])))
}

// execDdevCmd hands the terminal to the ddev subcommand of op. Its output
// goes to the terminal's normal screen, so it is still there, errors
// included, after the TUI quits.
func execDdevCmd(op operationAction) tea.Cmd {
	c, err := ddevCommand(op.dir, op.args...)
	if err != nil {
		return func() tea.Msg { return op.finished(err) }
	}
	return tea.Exec(terminalCommand{Cmd: c, pause: execPause}, op.finished)
}

// finished reports how the command of op exited.
func (op operationAction) finished(err error) tea.Msg {
	var exitErr *exec.ExitError
	if op.shell && errors.As(err, &exitErr) {
		return execFinishedMsg{title: op.title, done: fmt.Sprintf("%s (%v)", op.done, err)}
	}
	return execFinishedMsg{title: op.title, done: op.done, err: err, selectCurrent: op.selectCurrent}
}

// terminalCommand runs a ddev subcommand in the terminal. It prints the
// command first, so the terminal's scrollback shows what ran even when it
// prints nothing, and leaves the output up for pause after it exits.
type terminalCommand struct {
	*exec.Cmd
	pause time.Duration
}

func (c terminalCommand) Run() error {
	if c.Stdout != nil {
		_, _ = fmt.Fprintln(c.Stdout, shellLine(c.Dir, c.Args[1:]))
	}
	err := c.Cmd.Run()
	time.Sleep(c.pause)
	return err
}

// shellLine is a shell command that runs ddev with args in dir, or in the
// current directory when dir is empty.
func shellLine(dir string, args []string) string {
	line := "$ ddev " + strings.Join(args, " ")
	if dir == "" {
		return line
	}
	path := formatProjectPath(dir)
	if strings.ContainsFunc(path, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("~/._-+@:,", r)
	}) {
		// Quoted, ~ would not expand, so the full path is used.
		path = "'" + strings.ReplaceAll(filepath.ToSlash(dir), "'", `'\''`) + "'"
	}
	return "$ cd " + path + " && ddev " + strings.Join(args, " ")
}

// The setters hand the command the terminal's own files. A writer that is
// not a file would be copied through a pipe, so stderr could land after a
// prompt written to stdout later.
func (c terminalCommand) SetStdin(r io.Reader)  { c.Stdin = r }
func (c terminalCommand) SetStdout(w io.Writer) { c.Stdout = w }
func (c terminalCommand) SetStderr(w io.Writer) { c.Stderr = w }

// startStreamCmd starts a ddev subcommand in dir and streams its combined
// output as stream id.
func startStreamCmd(id int, dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		c, err := ddevCommand(dir, args...)
		if err != nil {
			return streamEndedMsg{id: id, err: err}
		}
		stdout, outW := io.Pipe()
		stderr, errW := io.Pipe()
		c.Stdout, c.Stderr = outW, errW
		if err := c.Start(); err != nil {
			return streamEndedMsg{id: id, err: err}
		}

		lines := make(chan string, 100)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup
		scan := func(r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				lines <- strings.TrimRight(scanner.Text(), "\r")
			}
			// A line too long to scan must not block the subprocess.
			_, _ = io.Copy(io.Discard, r)
		}
		wg.Add(2)
		go scan(stdout)
		go scan(stderr)
		go func() {
			err := c.Wait()
			_ = outW.Close()
			_ = errW.Close()
			wg.Wait()
			errCh <- err
			// Closed so a second reader after the exit error gets nil
			// instead of blocking.
			close(errCh)
			close(lines)
		}()
		return streamStartedMsg{id: id, lines: lines, errCh: errCh, process: c.Process}
	}
}

// waitForStreamCmd waits for output from stream id, or for its exit. It
// takes every line already waiting, so a burst of output is one update.
func waitForStreamCmd(id int, lines <-chan string, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if !ok {
			return streamEndedMsg{id: id, err: <-errCh}
		}
		batch := []string{line}
		for len(batch) < 1000 {
			select {
			case line, ok := <-lines:
				if !ok {
					return streamLinesMsg{id: id, lines: batch}
				}
				batch = append(batch, line)
			default:
				return streamLinesMsg{id: id, lines: batch}
			}
		}
		return streamLinesMsg{id: id, lines: batch}
	}
}

func latestAddonsCmd(appRoot string) tea.Cmd {
	return func() tea.Msg {
		return latestAddonsMsg{appRoot: appRoot, versions: latestAddonVersions()}
	}
}

// openTarget hands a directory path or URL to the OS opener without taking
// over the terminal. Editors are reached through their URL schemes so no
// editor command has to be on PATH.
func openTarget(what, target string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch {
		case nodeps.IsMacOS():
			cmd = exec.Command("open", target)
		case nodeps.IsWindows():
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
		case nodeps.IsWSL2():
			return openedMsg{what: what, err: openOnWSL(target)}
		default:
			if _, err := exec.LookPath("xdg-open"); err != nil {
				return openedMsg{what: what, err: fmt.Errorf("no opener found (install xdg-utils)")}
			}
			cmd = exec.Command("xdg-open", target)
		}
		return openedMsg{what: what, err: cmd.Run()}
	}
}

// openInPhpStorm opens dir with a PhpStorm command-line launcher when one is
// on PATH, since PhpStorm does not register phpstorm:// on Windows.
func openInPhpStorm(dir string) tea.Cmd {
	var launcher string
	// Toolbox names it phpstorm on Linux and macOS, PhpStorm on Windows
	for _, name := range []string{"phpstorm", "PhpStorm", "phpstorm64.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			launcher = p
			break
		}
	}
	if launcher == "" {
		return openTarget("PhpStorm", editorURL("phpstorm", dir))
	}
	return func() tea.Msg {
		if nodeps.IsWSL2() {
			winPath, err := exec.Command("wslpath", "-w", dir).Output()
			if err != nil {
				return openedMsg{what: "PhpStorm", err: fmt.Errorf("wslpath -w %s: %w", dir, err)}
			}
			dir = strings.TrimSpace(string(winPath))
		}
		// The launcher may run as the IDE process itself, so don't wait for it
		c := exec.Command(launcher, dir)
		if err := c.Start(); err != nil {
			return openedMsg{what: "PhpStorm", err: err}
		}
		go func() { _ = c.Wait() }()
		return openedMsg{what: "PhpStorm"}
	}
}

// openOnWSL opens target with Windows tools directly, since xdg-open's WSL
// support is missing before xdg-utils 1.2 and reports failure whenever it
// opens a directory.
func openOnWSL(target string) error {
	if strings.Contains(target, "://") {
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target).Run()
	}
	winPath, err := exec.Command("wslpath", "-w", target).Output()
	if err != nil {
		return fmt.Errorf("wslpath -w %s: %w", target, err)
	}
	// explorer.exe exits non-zero even on success
	_ = exec.Command("explorer.exe", strings.TrimSpace(string(winPath))).Run()
	return nil
}

// copyToClipboard copies text with the system clipboard tool. Without one,
// the clipboardMsg asks for the terminal's OSC 52 clipboard instead.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		if err := writeClipboard(text); err != nil {
			return clipboardMsg{fallback: text}
		}
		return clipboardMsg{}
	}
}
