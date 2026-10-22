package tui

import (
	"cmp"
	"fmt"
	"image/color"
	"os"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type viewMode int

const (
	viewDashboard viewMode = iota
	viewDetail
	viewStream
)

// AppModel is the root Bubble Tea model. It owns what spans the views: the
// window, the header and footer, the status line, confirmations, the shared
// containers' health and the refresh ticker. Each view owns its own state and
// replies to keys with an action that AppModel applies.
type AppModel struct {
	keys   KeyMap
	styles Styles
	// fg and bg are the terminal's text and background colors, once known.
	fg, bg  color.Color
	help    help.Model
	spinner spinner.Model
	width   int
	height  int
	view    viewMode

	dashboard dashboardView
	detail    detailView
	stream    streamView
	streamID  int

	// refreshing is set while a reload the user asked for runs.
	refreshing bool
	router     health
	sshAgent   health

	confirm     *pendingConfirm
	status      string
	statusIsErr bool
	statusID    int
	// lastWarning is what the latest load logged.
	lastWarning string
}

// NewAppModel creates a new TUI model.
func NewAppModel() AppModel {
	styles := NewStyles(nil, nil)
	h := help.New()
	h.Styles = styles.help()
	cwd, _ := os.Getwd()
	return AppModel{
		keys:      DefaultKeyMap(),
		styles:    styles,
		help:      h,
		spinner:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		dashboard: newDashboardView(styles, cwd),
		router:    health{status: statusLoading},
		sshAgent:  health{status: statusLoading},
	}
}

// Init starts the first load and the refresh ticker. The configuration of
// the projects is read first, since it shows without waiting for Docker.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tea.Sequence(readProjects, loadProjects), tickCmd(), m.spinner.Tick, tea.RequestForegroundColor, tea.RequestBackgroundColor)
}

// Update handles a message, then lays out the views for the result, so a
// change in the size of the header or footer resizes the body at once.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	wasBusy := m.busy()
	m, cmd := m.update(msg)
	// Only what View renders shows the spinner, and a frame adds no line.
	if _, tick := msg.(spinner.TickMsg); !tick {
		m.layout()
	}
	if !wasBusy && m.busy() {
		cmd = tea.Batch(cmd, m.spinner.Tick)
	}
	return m, cmd
}

func (m AppModel) update(msg tea.Msg) (AppModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.ForegroundColorMsg:
		m.fg = msg.Color
		return m.restyle(), nil

	case tea.BackgroundColorMsg:
		m.bg = msg.Color
		return m.restyle(), nil

	case tea.KeyPressMsg:
		return m.updateKey(msg)

	case spinner.TickMsg:
		if !m.busy() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		if m.view == viewStream || m.dashboard.loading {
			return m, tickCmd()
		}
		m.dashboard.loading = true
		return m, tea.Batch(loadProjects, tickCmd())

	case projectsListedMsg:
		m.dashboard.setProjects(msg.projects)
		return m, nil

	case projectsLoadedMsg:
		m, logged := m.logged(msg.warning)
		m, cmd := m.projectsLoaded(msg)
		return m, tea.Batch(logged, cmd)

	case projectDetailLoadedMsg:
		m, logged := m.logged(msg.warning)
		m, cmd := m.detailLoaded(msg)
		return m, tea.Batch(logged, cmd)

	case xdebugStatusMsg:
		var act action
		m.detail, act = m.detail.xdebugLoaded(msg)
		return m.apply(act)

	case latestAddonsMsg:
		if msg.appRoot == m.detail.root {
			m.detail.latest = msg.versions
		}
		return m, nil

	case statusExpiredMsg:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil

	case execFinishedMsg:
		if msg.err != nil {
			m = m.setError(fmt.Sprintf("%s failed: %v", msg.title, msg.err))
			return m.reload()
		}
		if msg.selectCurrent {
			m.dashboard.selectCurrent = true
		}
		m, cmd := m.setStatus(cmp.Or(msg.done, msg.title+": done"))
		m, reload := m.reload()
		return m, tea.Batch(cmd, reload)

	case openedMsg:
		if msg.err != nil {
			return m.setError(fmt.Sprintf("Open %s failed: %v", msg.what, msg.err)), nil
		}
		return m.setStatus("Opened " + msg.what)

	case clipboardMsg:
		if msg.fallback != "" {
			m, cmd := m.setStatus("Sent the URL to the terminal's clipboard")
			return m, tea.Batch(cmd, tea.SetClipboard(msg.fallback))
		}
		return m.setStatus("Copied the URL to the clipboard")

	case streamStartedMsg, streamLinesMsg, streamEndedMsg:
		var cmd tea.Cmd
		m.stream, cmd = m.stream.update(msg)
		return m, cmd
	}

	if m.view == viewDashboard {
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.updateFilter(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) restyle() AppModel {
	m.styles = NewStyles(m.fg, m.bg)
	m.help.Styles = m.styles.help()
	m.dashboard.setStyles(m.styles)
	return m
}

func (m AppModel) updateKey(msg tea.KeyPressMsg) (AppModel, tea.Cmd) {
	// A key means the error was seen.
	if m.statusIsErr {
		m.status, m.statusIsErr = "", false
	}
	if msg.String() == "ctrl+c" {
		return m.quit()
	}
	if m.confirm != nil {
		return m.handleConfirm(msg)
	}
	if m.view != viewDashboard || !m.dashboard.filtering() {
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case m.help.ShowAll && msg.String() == "esc":
			m.help.ShowAll = false
			return m, nil
		case key.Matches(msg, m.keys.Quit):
			return m.quit()
		}
	}
	var cmd tea.Cmd
	var act action
	switch m.view {
	case viewDetail:
		m.detail, cmd, act = m.detail.update(msg, m.keys)
	case viewStream:
		m.stream, act = m.stream.updateKey(msg, m.keys)
	default:
		m.dashboard, cmd, act = m.dashboard.update(msg, m.keys)
	}
	m, actCmd := m.apply(act)
	return m, tea.Batch(cmd, actCmd)
}

func (m AppModel) quit() (AppModel, tea.Cmd) {
	m, _ = m.stopStream()
	return m, tea.Quit
}

// busy reports whether anything shows the spinner.
func (m AppModel) busy() bool {
	return m.dashboard.waiting() || m.refreshing || m.detail.loading || m.streaming()
}

// streaming reports whether the logs are still followed.
func (m AppModel) streaming() bool {
	return m.view == viewStream && !m.stream.done && m.stream.follow
}

// reload refreshes the project list, and the detail when it is showing or
// the stream returns to it.
func (m AppModel) reload() (AppModel, tea.Cmd) {
	cmds := []tea.Cmd{}
	if m.dashboard.loading {
		// The running load may have read the projects before what asked for
		// this reload, such as an operation, changed them.
		m.dashboard.queued = true
	} else {
		m.dashboard.loading = true
		cmds = append(cmds, loadProjects)
	}
	if m.view == viewDetail {
		// What ran, such as a restart or ssh, may have switched Xdebug.
		m.detail.loading, m.detail.xdebug = true, nil
		cmds = append(cmds, loadDetailCmd(m.detail.root))
	}
	return m, tea.Batch(cmds...)
}

// refresh is a reload the user asked for, so it shows the spinner.
func (m AppModel) refresh() (AppModel, tea.Cmd) {
	m.refreshing = true
	return m.reload()
}

func (m AppModel) projectsLoaded(msg projectsLoadedMsg) (AppModel, tea.Cmd) {
	m, cmd := m.showProjects(msg)
	if m.dashboard.queued {
		m.dashboard.queued, m.dashboard.loading = false, true
		return m, tea.Batch(cmd, loadProjects)
	}
	m.refreshing = false
	return m, cmd
}

func (m AppModel) showProjects(msg projectsLoadedMsg) (AppModel, tea.Cmd) {
	m.dashboard.loading = false
	if msg.err != nil {
		// Until a load succeeds, the error also stands in for the spinner.
		if !m.dashboard.loaded || len(m.dashboard.items) == 0 {
			m.dashboard.err = msg.err
			m.dashboard.markUnknown()
			m.router, m.sshAgent = health{status: statusUnknown}, health{status: statusUnknown}
		}
		if len(m.dashboard.items) > 0 {
			return m.setError(fmt.Sprintf("Error loading projects: %v", msg.err)), nil
		}
		return m, nil
	}
	m.dashboard.err = nil
	m.dashboard.loaded = true
	m.router, m.sshAgent = msg.router, msg.sshAgent
	m.dashboard.setProjects(msg.projects)

	// The 5-second refresh only reloads the detail when its project changed
	// state, because Describe inspects every container.
	if m.view == viewDetail && m.detail.project != nil && !m.detail.loading && m.detail.statusChanged(msg.projects) {
		m.detail.loading = true
		return m, loadDetailCmd(m.detail.root)
	}
	return m, nil
}

// openDetail shows the detail of project p and starts loading it.
func (m AppModel) openDetail(p ProjectInfo) (AppModel, tea.Cmd) {
	m.view = viewDetail
	m.detail = newDetailView(p)
	m.help.ShowAll = false
	return m, loadDetailCmd(p.AppRoot)
}

func (m AppModel) detailLoaded(msg projectDetailLoadedMsg) (AppModel, tea.Cmd) {
	if msg.appRoot != m.detail.root {
		return m, nil
	}
	if msg.err != nil {
		m.detail.loading = false
		if (m.detail.project == nil || m.detail.project.FromConfig) && m.view == viewDetail {
			m.view = viewDashboard
		}
		return m.setError(fmt.Sprintf("Error loading %s: %v", m.detail.name, msg.err)), nil
	}
	var cmd tea.Cmd
	m.detail, cmd = m.detail.loaded(msg.detail)
	return m, cmd
}

// newStream shows a new stream, which returns to the current view when left.
func (m AppModel) newStream(title string) AppModel {
	m.streamID++
	m.stream = newStreamView(m.streamID, title, m.view)
	m.view = viewStream
	m.help.ShowAll = false
	m.status = ""
	return m
}

// stopStream kills the stream's subprocess and leaves the stream view. The
// returned command drains what the subprocess already wrote, so its reader
// goroutines can finish.
func (m AppModel) stopStream() (AppModel, tea.Cmd) {
	s := m.stream
	s.kill()
	if m.view == viewStream {
		m.view = s.returnView
	}
	m.stream = streamView{}
	return m, drainCmd(s.lineCh)
}

// setStatus shows an informational message that clears itself.
func (m AppModel) setStatus(text string) (AppModel, tea.Cmd) {
	m.statusID++
	m.status, m.statusIsErr = text, false
	return m, expireStatusCmd(m.statusID)
}

// logged shows what ddevapp logged while loading, such as a project removed
// from the registry. It asks nothing of the user, so it clears itself and
// leaves an error in place. Some warnings recur on every refresh, so one
// shows again only after a load without it.
func (m AppModel) logged(text string) (AppModel, tea.Cmd) {
	last := m.lastWarning
	m.lastWarning = text
	if text == "" || text == last || (m.statusIsErr && m.status != "") {
		return m, nil
	}
	return m.setStatus(text)
}

// setError shows an error, which stays until another message replaces it.
func (m AppModel) setError(text string) AppModel {
	m.statusID++
	m.status, m.statusIsErr = text, true
	return m
}
