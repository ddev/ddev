package tui

import (
	"math"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
)

// ui is what the app shares with its views for rendering and key handling.
type ui struct {
	styles  Styles
	keys    KeyMap
	width   int
	spinner string
}

func (m AppModel) ui() ui {
	return ui{styles: m.styles, keys: m.keys, width: m.width, spinner: m.spinner.View()}
}

// helpBinding is the help key, described by what pressing it does now.
func (m AppModel) helpBinding() key.Binding {
	b := m.keys.Help
	if m.help.ShowAll {
		b.SetHelp(b.Help().Key, "close help")
	}
	return b
}

func (m AppModel) helpKeys() help.KeyMap {
	switch {
	case m.confirm != nil:
		return m.confirmHelp()
	case m.view == viewDetail:
		return m.detail.help(m.keys, m.helpBinding())
	case m.view == viewStream:
		return m.stream.help(m.keys, m.helpBinding())
	}
	return m.dashboard.help(m.keys, m.helpBinding())
}

// layout sizes the active view to the space between the header and footer.
func (m *AppModel) layout() {
	m.help.SetWidth(m.width)
	h := m.bodyHeight()
	// Every view keeps the last column for its scrollbar.
	w := max(1, m.width-1)
	switch m.view {
	case viewDashboard:
		m.dashboard.resize(w, h)
	case viewDetail:
		m.detail.resize(m.styles, w, h)
	case viewStream:
		m.stream.resize(w, h)
	}
}

func (m AppModel) bodyHeight() int {
	return max(1, m.height-lipgloss.Height(m.header())-lipgloss.Height(m.footer()))
}

// header is the active view's title, the shared containers' health on the
// right, and a divider.
func (m AppModel) header() string {
	u := m.ui()
	title, busy := m.dashboard.title(u), m.refreshing || m.dashboard.waiting()
	switch m.view {
	case viewDetail:
		title, busy = m.detail.title(u), busy || m.detail.loading
	case viewStream:
		title, busy = m.stream.title(u), !m.stream.done
	}
	if busy {
		title += " " + u.spinner
	}
	health := m.health(false)
	if ansi.StringWidth(title)+1+ansi.StringWidth(health) > m.width {
		health = m.health(true)
	}
	return headerBar(title, health, m.width) + "\n" + m.divider()
}

func (m AppModel) body() string {
	u := m.ui()
	switch m.view {
	case viewDetail:
		return m.detail.body(u)
	case viewStream:
		return scrollable(m.styles, m.stream.vp)
	}
	return m.dashboard.body(u)
}

// footer holds the status line, the shared containers' problems and the help.
func (m AppModel) footer() string {
	lines := []string{m.divider()}
	// A long message would push the help off screen, so it is cut, leaving a
	// question an extra line.
	if s := m.statusLine(); s != "" {
		n := 2
		if m.confirm != nil {
			n = 3
		}
		lines = append(lines, clip(m.width, s, n))
	}
	lines = append(lines, m.healthProblems()...)
	// The stream keeps its line even when empty, so following or pausing
	// does not resize the body it depends on.
	if m.view == viewStream {
		lines = append(lines, m.stream.info(m.styles, m.keys))
	}
	return strings.Join(append(lines, m.helpView()), "\n")
}

// statusLine is the pending confirmation, the latest status message, or a
// hint for the current view, in that order.
func (m AppModel) statusLine() string {
	switch {
	case m.confirm != nil:
		return m.styles.Prompt.Render(m.confirm.prompt)
	case m.status != "" && m.statusIsErr:
		return m.styles.Error.Render(m.status)
	case m.status != "":
		return m.styles.Info.Render(m.status)
	case m.view == viewDashboard && m.dashboard.hint(m.keys) != "":
		return m.styles.Info.Render(m.dashboard.hint(m.keys))
	}
	return ""
}

// helpView renders the help, falling back to the short help when the full
// help would leave no room for the body.
func (m AppModel) helpView() string {
	h := m.help
	if h.ShowAll {
		full := h.View(m.helpKeys())
		// 6 is the two-line header, the footer's divider and three body rows.
		if m.height == 0 || lipgloss.Height(full)+6 <= m.height {
			return full
		}
		h.ShowAll = false
	}
	return h.View(m.helpKeys())
}

// health is the ssh-agent and router status, for the header. Compact, it
// shows only their symbols.
func (m AppModel) health(compact bool) string {
	var parts []string
	add := func(name, short, status string, fault bool) {
		if status == "" {
			return
		}
		symbol, style := m.styles.statusLook(status)
		if fault {
			symbol, style = "✗", m.styles.Error
		}
		text := symbol + " " + statusLabel(status)
		if compact {
			name, text = short, symbol
		}
		parts = append(parts, m.styles.Label.Render(name+" ")+style.Render(text))
	}
	add("SSH agent", "SSH", m.sshAgent.status, false)
	// DDEV removes the router when no project runs, so stopped is a fault
	// only while one does.
	add("Router", "Router", m.router.status, m.router.status == ddevapp.SiteStopped && m.dashboard.runningCount() > 0)
	return strings.Join(parts, m.styles.Muted.Render(" · "))
}

// healthProblems is what a failing router or ssh-agent healthcheck reported.
func (m AppModel) healthProblems() []string {
	var lines []string
	for _, p := range []struct {
		name string
		h    health
	}{{"Router", m.router}, {"SSH agent", m.sshAgent}} {
		if problem := p.h.problem(); problem != "" {
			lines = append(lines, clip(m.width, m.styles.Error.Render(p.name+": "+problem), 2))
		}
	}
	return lines
}

func (m AppModel) divider() string {
	return m.styles.Muted.Render(strings.Repeat("─", max(1, m.width)))
}

// View renders the TUI on the alternate screen, cut to the window size.
func (m AppModel) View() tea.View {
	bodyH := m.bodyHeight()
	body := lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(m.body())
	content := m.header() + "\n" + body + "\n" + m.footer()
	if m.width > 0 && m.height > 0 {
		content = lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "DDEV"
	return v
}

// scrollable renders vp with a scrollbar in the column to its right, which
// the views keep free so that the bar appearing does not rewrap the content.
func scrollable(s Styles, vp viewport.Model) string {
	lines := strings.Split(vp.View(), "\n")
	total, h := vp.TotalLineCount(), len(lines)
	thumb, top := h, 0
	if total > h {
		thumb = max(1, h*h/total)
		top = int(math.Round(vp.ScrollPercent() * float64(h-thumb)))
	}
	for i, line := range lines {
		bar := " "
		switch {
		case total <= h:
		case i >= top && i < top+thumb:
			bar = s.Selected.Render("┃")
		default:
			bar = s.Muted.Render("│")
		}
		lines[i] = fit(line, vp.Width()) + bar
	}
	return strings.Join(lines, "\n")
}

// newViewport returns a viewport with the TUI's scroll keys.
func newViewport() viewport.Model {
	vp := viewport.New()
	vp.KeyMap = viewportKeyMap()
	return vp
}

// clip wraps s to width and keeps at most n lines, ending the last kept line
// with an ellipsis when more were cut.
func clip(width int, s string, n int) string {
	w := max(1, width)
	lines := strings.Split(ansi.Wrap(s, w, ""), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	lines = lines[:n]
	lines[n-1] = ansi.Truncate(lines[n-1], w-1, "") + "…"
	return strings.Join(lines, "\n")
}

// message renders text for a view's body, indented and wrapped to width.
func message(width int, lines ...string) string {
	return lipgloss.NewStyle().PaddingLeft(2).Width(max(3, width)).Render(strings.Join(lines, "\n"))
}

// headerBar puts left and right on one line of width, dropping right when
// they do not fit.
func headerBar(left, right string, width int) string {
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}
