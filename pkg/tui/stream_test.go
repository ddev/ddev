package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// streamModel returns a model following the logs of a project, with the
// subprocess started.
func streamModel(t *testing.T, width, height int) AppModel {
	t.Helper()
	stubCommands(t)
	m := keys(t, detailModel(t, width, height, sampleDetail()), "L")
	return send(t, m, streamStartedMsg{id: m.stream.id, lines: make(chan string), errCh: make(chan error)})
}

func lines(id, from, to int) streamLinesMsg {
	msg := streamLinesMsg{id: id}
	for i := from; i < to; i++ {
		msg.lines = append(msg.lines, fmt.Sprintf("line %d", i))
	}
	return msg
}

func TestStreamFollowsOutput(t *testing.T) {
	m := streamModel(t, 80, 12)
	require.Contains(t, screen(m), "Waiting for output...")
	m = send(t, m, lines(m.stream.id, 0, 50))
	require.Contains(t, screen(m), "line 49", "new output is followed")
	require.NotContains(t, screen(m), "Waiting for output")

	m = keys(t, m, "up")
	require.False(t, m.stream.follow)
	require.Contains(t, screen(m), "Paused, press G/end to follow new output")
	m = send(t, m, lines(m.stream.id, 50, 60))
	require.NotContains(t, screen(m), "line 59", "scrolled back output stays put")
	require.Contains(t, screen(m), "line 48")

	m = keys(t, m, "end")
	require.True(t, m.stream.follow)
	require.Contains(t, screen(m), "line 59")
	m = send(t, m, lines(m.stream.id, 60, 61))
	require.Contains(t, screen(m), "line 60")

	m = keys(t, m, "g")
	require.Contains(t, screen(m), "line 0")
	require.Equal(t, viewStream, keys(t, m, "l", "d", "s").view, "action keys do nothing here")
}

func TestStreamWrapsAndCaps(t *testing.T) {
	m := streamModel(t, 40, 10)
	long := strings.Repeat("x", 100) + "END"
	m = send(t, m, streamLinesMsg{id: m.stream.id, lines: []string{"\tindented", long}})
	require.Contains(t, screen(m), "END", "a long line wraps instead of being cut")
	require.Contains(t, screen(m), "    indented", "tabs, which have no width, become spaces")
	requireFits(t, m, "wrapped")

	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 10})
	require.Contains(t, screen(m), long, "a wider window unwraps it")

	m = send(t, m, lines(m.stream.id, 0, maxStreamLines+10))
	require.LessOrEqual(t, len(m.stream.lines), maxStreamLines)
	require.Equal(t, fmt.Sprintf("line %d", maxStreamLines+9), m.stream.lines[len(m.stream.lines)-1])
	require.Contains(t, screen(m), fmt.Sprintf("line %d", maxStreamLines+9))
}

func TestStreamCleansOutput(t *testing.T) {
	m := streamModel(t, 40, 12)
	m = send(t, m, streamLinesMsg{id: m.stream.id, lines: []string{
		"Pulling 10%\rPulling 50%\rPulling 100%",
		"\x1b[1A\x1b[2Kdone\x07\x1b[?25l",
		"\x1b[33mWarning: from a hook",
		"still yellow " + strings.Repeat("y", 40) + "\x1b[0m",
		"plain",
		"\x1b]8;;https://example.com\x07link\x1b]8;;\x07",
	}})
	require.Equal(t, []string{"Pulling 100%", "done", "\x1b[33mWarning: from a hook", "still yellow " + strings.Repeat("y", 40) + "\x1b[0m", "plain", "link"}, m.stream.lines,
		"a carriage return overwrites the line, and only colors are kept")

	colors := regexp.MustCompile(`\x1b\[[0-9;:]*m`)
	for _, line := range m.stream.wrapped {
		require.False(t, strings.ContainsFunc(colors.ReplaceAllString(line, ""), isControl), "%q", line)
	}
	// The warning spans two lines, the second of which wraps.
	require.Len(t, m.stream.wrapped, 7)
	for _, line := range m.stream.wrapped[2:5] {
		require.Regexp(t, `^\x1b\[33m.*\x1b\[0?m$`, line, "each line of the warning is yellow on its own")
	}
	require.Equal(t, "plain", m.stream.wrapped[5], "the color ends where the output reset it")
	requireFits(t, m, "cleaned")
}

func TestStreamColorsDoNotPileUp(t *testing.T) {
	m := streamModel(t, 80, 12)
	var out []string
	for range 100 {
		out = append(out, "\x1b[32mInstalling\x1b[39m package")
	}
	out = append(out, "\x1b[1;33mWarning", "bold and yellow\x1b[22m", "yellow\x1b[39m", "plain")
	m = send(t, m, streamLinesMsg{id: m.stream.id, lines: out})
	w := m.stream.wrapped
	require.Equal(t, "\x1b[32mInstalling\x1b[39m package", w[99], "a color reset by 39 is not carried to the next line")
	require.True(t, strings.HasPrefix(w[101], "\x1b[1;33mbold"), "%q", w[101])
	require.True(t, strings.HasPrefix(w[102], "\x1b[33myellow"), "22 ends the bold but keeps the color: %q", w[102])
	require.Equal(t, "plain", w[103])
}

func TestStreamIgnoresStaleMessages(t *testing.T) {
	m := streamModel(t, 80, 12)
	old := m.stream.id
	m = keys(t, m, "esc")
	require.Equal(t, viewDetail, m.view)
	m = send(t, m, lines(old, 0, 3), streamEndedMsg{id: old})
	require.Equal(t, viewDetail, m.view)
	require.Empty(t, m.stream.lines)

	m = keys(t, m, "L")
	m = send(t, m, lines(old, 0, 3))
	require.Empty(t, m.stream.lines, "output of the earlier stream does not show in a new one")
}

func TestLogsEnd(t *testing.T) {
	stubCommands(t)
	m := keys(t, detailModel(t, 80, 24, sampleDetail()), "L")
	m = send(t, m, streamStartedMsg{id: m.stream.id, lines: make(chan string), errCh: make(chan error)}, streamEndedMsg{id: m.stream.id})
	require.Equal(t, viewStream, m.view, "logs never return by themselves")
	require.Contains(t, screen(m), "Log stream ended")
	require.Equal(t, viewDetail, keys(t, m, "esc").view)
}

func TestTerminalCommand(t *testing.T) {
	useFakeDdev(t)
	fakeOutput(t, "started\n", "", true, "start", "alpha")
	dir := t.TempDir()
	c, err := ddevCommand(dir, "start", "alpha")
	require.NoError(t, err)
	var out bytes.Buffer
	cmd := terminalCommand{Cmd: c, pause: 50 * time.Millisecond}
	cmd.SetStdout(&out)
	begin := time.Now()
	require.Error(t, cmd.Run(), "a failure is reported")
	require.GreaterOrEqual(t, time.Since(begin), cmd.pause, "the output stays up before the TUI returns")
	require.Equal(t, shellLine(dir, []string{"start", "alpha"})+"\nstarted\n", out.String(), "the command shows above its output")

	for _, shell := range []bool{false, true} {
		c, err = ddevCommand(dir, "start", "alpha")
		require.NoError(t, err)
		out.Reset()
		cmd = terminalCommand{Cmd: c, shell: shell}
		cmd.SetStdin(strings.NewReader("\n"))
		cmd.SetStdout(&out)
		require.Error(t, cmd.Run())
		require.Equal(t, !shell, strings.Contains(out.String(), "Press Enter"), "a shell's exit status is not a failure")
	}
}

func TestShellExitStatus(t *testing.T) {
	useFakeDdev(t)
	fakeOutput(t, "", "", true, "ssh")
	c, err := ddevCommand("", "ssh")
	require.NoError(t, err)
	exitErr := c.Run()
	require.Error(t, exitErr)

	ssh := operationAction{title: "SSH into alpha", done: "SSH session ended", shell: true}
	require.Equal(t, execFinishedMsg{title: ssh.title, done: "SSH session ended (exit status 1)"}, ssh.finished(exitErr),
		"the shell exits with the status of the user's last command")
	require.Equal(t, execFinishedMsg{title: ssh.title, done: ssh.done, err: os.ErrNotExist}, ssh.finished(os.ErrNotExist),
		"a shell that could not run is a failure")
	start := operation("Starting alpha", "", "start")
	require.Equal(t, execFinishedMsg{title: start.title, err: exitErr}, start.finished(exitErr))
}

func TestShellLine(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.Equal(t, "$ ddev start --all", shellLine("", []string{"start", "--all"}))
	require.Equal(t, "$ cd ~/apps/my-site_2\n$ ddev launch", shellLine(filepath.Join(home, "apps", "my-site_2"), []string{"launch"}))
	require.Equal(t, "$ cd '/srv/it'\\''s here'\n$ ddev launch", shellLine("/srv/it's here", []string{"launch"}), "quoted, without ~, which would not expand")
}

// collectStream runs the stream's commands until it ends.
func collectStream(t *testing.T, cmd tea.Cmd) (got []string, err error) {
	t.Helper()
	started, ok := cmd().(streamStartedMsg)
	require.True(t, ok)
	wait := waitForStreamCmd(started.id, started.lines, started.errCh)
	for {
		switch msg := wait().(type) {
		case streamLinesMsg:
			got = append(got, msg.lines...)
		case streamEndedMsg:
			return got, msg.err
		}
	}
}

func TestStartStreamCmd(t *testing.T) {
	useFakeDdev(t)
	fakeOutput(t, "one\r\ntwo\n", "three\n", true, "start")
	got, err := collectStream(t, startStreamCmd(1, t.TempDir(), "start"))
	sort.Strings(got)
	require.Equal(t, []string{"one", "three", "two"}, got, "stdout and stderr, without carriage returns")
	require.Error(t, err)

	fakeOutput(t, "ok\n", "", false, "stop")
	_, err = collectStream(t, startStreamCmd(2, t.TempDir(), "stop"))
	require.NoError(t, err)
}

func TestLeavingStreamKillsSubprocess(t *testing.T) {
	useFakeDdev(t)
	t.Setenv("TUI_TEST_HANG_"+fakeKey([]string{"logs", "-f"}), "1")
	stubCommands(t)

	m := keys(t, detailModel(t, 80, 24, sampleDetail()), "L")
	started, ok := startStreamCmd(m.stream.id, t.TempDir(), "logs", "-f")().(streamStartedMsg)
	require.True(t, ok)
	m = send(t, m, started)
	m, cmd := sendCmd(t, m, press("esc"))
	require.Equal(t, viewDetail, m.view)
	requireFinishes(t, cmd, "leaving the logs kills ddev logs -f")

	late, ok := startStreamCmd(99, t.TempDir(), "logs", "-f")().(streamStartedMsg)
	require.True(t, ok)
	_, cmd = sendCmd(t, m, late)
	requireFinishes(t, cmd, "a subprocess that starts after its view was left is killed")
}

// requireFinishes runs cmd, including any batch it returns, and fails unless
// it finishes well before the hanging fake ddev would exit by itself.
func requireFinishes(t *testing.T, cmd tea.Cmd, msg string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		runAll(cmd)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		require.Fail(t, msg)
	}
}

func runAll(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			runAll(c)
		}
	}
}
