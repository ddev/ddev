package tui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// streamView follows the output of `ddev logs -f`.
type streamView struct {
	id         int
	name       string
	returnView viewMode
	lines      []string
	// wrapped is lines wrapped to wrapWidth, as the viewport shows them.
	wrapped   []string
	wrapWidth int
	// pen carries the colors still active after a wrapped line to the next;
	// output can leave them open over several lines. penOut is what it wrote.
	pen     *lipgloss.WrapWriter
	penOut  *bytes.Buffer
	process *os.Process
	lineCh  <-chan string
	errCh   <-chan error
	done    bool
	err     error
	// follow keeps the view at the newest line until the user scrolls up.
	follow bool
	vp     viewport.Model
}

func newStreamView(id int, name string, returnView viewMode) streamView {
	return streamView{id: id, name: name, returnView: returnView, follow: true, vp: newViewport()}
}

// kill stops the subprocess, if it is still running.
func (s streamView) kill() {
	if s.process != nil && !s.done {
		_ = s.process.Kill()
	}
}

// drainCmd reads the rest of an abandoned stream.
func drainCmd(lines <-chan string) tea.Cmd {
	if lines == nil {
		return nil
	}
	return func() tea.Msg {
		for range lines {
		}
		return nil
	}
}

// update handles the stream's messages.
func (s streamView) update(msg tea.Msg) (streamView, tea.Cmd) {
	switch msg := msg.(type) {
	case streamStartedMsg:
		if msg.id != s.id {
			// The user left before the subprocess started.
			_ = msg.process.Kill()
			return s, drainCmd(msg.lines)
		}
		s.process = msg.process
		s.lineCh = msg.lines
		s.errCh = msg.errCh
		return s, waitForStreamCmd(msg.id, msg.lines, msg.errCh)

	case streamLinesMsg:
		if msg.id != s.id {
			return s, nil
		}
		s.append(msg.lines...)
		if s.lineCh == nil {
			return s, nil
		}
		return s, waitForStreamCmd(msg.id, s.lineCh, s.errCh)

	case streamEndedMsg:
		if msg.id == s.id {
			s.done, s.err = true, msg.err
			s.process, s.lineCh = nil, nil
		}
	}
	return s, nil
}

// append adds lines of output. Lines are wrapped once here rather than by
// the viewport, whose soft wrap measures every line on each scroll.
func (s *streamView) append(lines ...string) {
	for _, line := range lines {
		line = cleanLine(line)
		s.lines = append(s.lines, line)
		s.wrapped = append(s.wrapped, s.wrap(line)...)
	}
	if len(s.lines) > maxStreamLines {
		s.lines = slices.Clone(s.lines[len(s.lines)-maxStreamLines/2:])
		s.rewrap()
		return
	}
	s.setContent()
}

func (s *streamView) resize(width, height int) {
	s.vp.SetWidth(width)
	s.vp.SetHeight(height)
	if width != s.wrapWidth {
		s.wrapWidth = width
		s.rewrap()
	}
	if s.follow {
		s.vp.GotoBottom()
	}
}

func (s *streamView) rewrap() {
	if s.pen != nil {
		_ = s.pen.Close()
	}
	s.wrapped, s.pen = nil, nil
	for _, line := range s.lines {
		s.wrapped = append(s.wrapped, s.wrap(line)...)
	}
	s.setContent()
}

// setContent gives the viewport, which only scrolls, a blank line for each
// wrapped one, since it measures every line it gets on every batch.
func (s *streamView) setContent() {
	s.vp.SetContentLines(make([]string, len(s.wrapped)))
	if s.follow {
		s.vp.GotoBottom()
	}
}

// wrap wraps line to wrapWidth. Each wrapped line reopens the colors still
// active from the lines before it and resets them at its end: the renderer
// redraws lines one at a time, so an open color would leak into the rest of
// the screen.
func (s *streamView) wrap(line string) []string {
	if s.wrapWidth > 0 {
		line = ansi.Hardwrap(line, s.wrapWidth, true)
	}
	if s.pen == nil {
		s.penOut = &bytes.Buffer{}
		s.pen = lipgloss.NewWrapWriter(s.penOut)
	}
	// The pen reopens the colors after each newline, so what it wrote after
	// the last one starts the next line.
	_, _ = io.WriteString(s.pen, line+"\n")
	out := s.penOut.String()
	end := strings.LastIndexByte(out, '\n')
	s.penOut.Reset()
	s.penOut.WriteString(out[end+1:])
	return strings.Split(out[:end], "\n")
}

// cleanLine keeps what a terminal shows of an output line without moving
// the cursor: the text after its last carriage return, and its colors.
// Other control characters and escape sequences, such as the cursor
// movement of progress output, would draw outside the line.
func cleanLine(line string) string {
	line = strings.ReplaceAll(line, "\t", "    ")
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	if !strings.ContainsFunc(line, isControl) {
		return line
	}
	var b strings.Builder
	var state byte
	for len(line) > 0 {
		seq, _, n, newState := ansi.DecodeSequence(line, state, nil)
		state, line = newState, line[n:]
		if isColor(seq) || !strings.ContainsFunc(seq, isControl) {
			b.WriteString(seq)
		}
	}
	return b.String()
}

func isControl(r rune) bool {
	return r < 0x20 || (r >= 0x7f && r < 0xa0) || r == utf8.RuneError
}

// isColor reports whether seq sets colors or text attributes (SGR).
func isColor(seq string) bool {
	return strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m") &&
		strings.Trim(seq[2:len(seq)-1], "0123456789;:") == ""
}

func (s streamView) keys(k KeyMap) KeyMap {
	k.End.SetHelp(k.End.Help().Key, "follow")
	return k
}

func (s streamView) help(k KeyMap, helpKey key.Binding) helpKeys {
	k = s.keys(k)
	vp := s.vp.KeyMap
	return helpKeys{
		short: []key.Binding{k.Back, vp.Up, vp.PageUp, k.End, helpKey, k.Quit},
		full: [][]key.Binding{
			{vp.Up, vp.Down, vp.PageUp, vp.PageDown, k.Home, k.End},
			{k.Back, helpKey, k.Quit},
		},
	}
}

func (s streamView) updateKey(msg tea.KeyPressMsg, k KeyMap) (streamView, action) {
	k = s.keys(k)
	switch {
	case key.Matches(msg, k.Back):
		return s, leaveStreamAction{}
	case key.Matches(msg, k.Home):
		s.vp.GotoTop()
	case key.Matches(msg, k.End):
		s.vp.GotoBottom()
	default:
		s.vp, _ = s.vp.Update(msg)
	}
	s.follow = s.vp.AtBottom()
	return s, nil
}

func (s streamView) title(u ui) string {
	return u.styles.Title.Render(s.name)
}

// info reports how the stream ended, or whether new output is followed.
func (s streamView) info(styles Styles, k KeyMap) string {
	switch {
	case s.done && s.err != nil:
		return styles.Error.Render(fmt.Sprintf("Failed: %v", s.err))
	case s.done:
		return styles.Info.Render("Log stream ended")
	case len(s.lines) == 0:
		return styles.Info.Render("Waiting for output...")
	case !s.follow:
		return styles.Info.Render("Paused, press " + k.End.Help().Key + " to follow new output")
	}
	return ""
}
