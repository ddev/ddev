package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/versionconstants"
)

const refreshInterval = 5 * time.Second

// View mode constants.
const (
	viewDashboard = iota
	viewDetail
	viewLogs
	viewOperation
)

// AppModel is the root Bubble Tea model.
type AppModel struct {
	projects     []ProjectInfo
	cursor       int
	filterText   string
	filtering    bool
	showHelp     bool
	loading      bool
	err          error
	width        int
	height       int
	keys         KeyMap
	styles       Styles
	routerStatus string
	statusMsg    string

	// View navigation
	viewMode int

	// Detail view
	detail        *ProjectDetail
	detailLoading bool

	// Log streaming
	logLines   []string
	logProcess *os.Process
	logSub     <-chan string

	// Operation streaming
	operationName       string
	operationDone       bool
	operationErr        error
	operationReturnView int
	operationErrCh      <-chan error

	// Spinner
	spinner spinner.Model

	// Confirmation overlay
	confirming    bool
	confirmAction string // "start-all", "stop-all", or "poweroff"

	// Viewports for scrolling
	dashboardViewport viewport.Model
	detailViewport    viewport.Model
	viewportReady     bool
}

// NewAppModel creates a new TUI model.
func NewAppModel() AppModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return AppModel{
		keys:    DefaultKeyMap(),
		styles:  NewStyles(),
		loading: true,
		spinner: s,
	}
}

// Init starts the initial data load and refresh ticker.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		loadProjects,
		loadRouterStatus,
		tickCmd(),
		m.spinner.Tick,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

type tickMsg struct{}

// filteredProjects returns the projects matching the current filter.
func (m AppModel) filteredProjects() []ProjectInfo {
	if m.filterText == "" {
		return m.projects
	}
	filter := strings.ToLower(m.filterText)
	var result []ProjectInfo
	for _, p := range m.projects {
		if strings.Contains(strings.ToLower(p.Name), filter) ||
			strings.Contains(strings.ToLower(p.Type), filter) ||
			strings.Contains(strings.ToLower(p.Status), filter) {
			result = append(result, p)
		}
	}
	return result
}

// allStopped returns true if all projects are stopped.
func (m AppModel) allStopped() bool {
	for _, p := range m.projects {
		if p.Status != ddevapp.SiteStopped {
			return false
		}
	}
	return true
}

// selectedProject returns the currently selected project, or nil.
func (m AppModel) selectedProject() *ProjectInfo {
	filtered := m.filteredProjects()
	if len(filtered) == 0 || m.cursor >= len(filtered) {
		return nil
	}
	return &filtered[m.cursor]
}

// enterOperationView sets up the model for the operation streaming view.
func (m AppModel) enterOperationView(name string, returnView int) AppModel {
	m.viewMode = viewOperation
	m.logLines = nil
	m.logProcess = nil
	m.logSub = nil
	m.operationName = name
	m.operationDone = false
	m.operationErr = nil
	m.operationReturnView = returnView
	m.operationErrCh = nil
	m.statusMsg = ""
	return m
}

// Update handles messages and key presses.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.viewportReady {
			m.dashboardViewport = viewport.New(msg.Width, msg.Height-10)
			m.detailViewport = viewport.New(msg.Width, msg.Height-6)
			m.viewportReady = true
		} else {
			m.dashboardViewport.Width = msg.Width
			m.dashboardViewport.Height = msg.Height - 10
			m.detailViewport.Width = msg.Width
			m.detailViewport.Height = msg.Height - 6
		}
		return m, nil

	case projectsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Error loading projects: %v", msg.err)
			return m, nil
		}
		m.err = nil
		if m.statusMsg == "Refreshing..." {
			m.statusMsg = ""
		}
		m.projects = msg.projects
		// Keep cursor in bounds
		filtered := m.filteredProjects()
		if m.cursor >= len(filtered) {
			m.cursor = max(0, len(filtered)-1)
		}
		// Update viewport with dashboard content only if we're in dashboard view and viewport is ready
		if m.viewMode == viewDashboard && m.viewportReady {
			m.updateDashboardViewport()
		}
		return m, nil

	case projectDetailLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading detail: %v", msg.err)
			m.viewMode = viewDashboard
			return m, nil
		}
		if m.statusMsg == "Refreshing..." {
			m.statusMsg = ""
		}
		m.detail = &msg.detail
		// Update viewport with detail content
		if m.viewportReady {
			m.detailViewport.SetContent(m.buildDetailContent())
		}
		return m, nil

	case routerStatusMsg:
		m.routerStatus = msg.status
		return m, nil

	case xdebugToggledMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Xdebug toggle failed: %v", msg.err)
		} else if msg.enabled {
			m.statusMsg = "Xdebug enabled"
		} else {
			m.statusMsg = "Xdebug disabled"
		}
		// Reload detail to reflect new xdebug state
		if m.detail != nil {
			m.detailLoading = true
			return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), m.spinner.Tick)
		}
		return m, nil

	case clipboardMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Copy failed: %v", msg.err)
		} else {
			m.statusMsg = "URL copied to clipboard"
		}
		return m, nil

	case logStreamStartedMsg:
		m.logProcess = msg.process
		m.logSub = msg.lines
		return m, waitForLogLineCmd(m.logSub)

	case operationStreamStartedMsg:
		m.logProcess = msg.process
		m.logSub = msg.lines
		m.operationErrCh = msg.errCh
		return m, waitForOperationLineCmd(m.logSub, m.operationErrCh)

	case logLineMsg:
		m.logLines = append(m.logLines, msg.line)
		// Cap at 1000 lines to prevent unbounded growth
		if len(m.logLines) > 1000 {
			m.logLines = m.logLines[len(m.logLines)-500:]
		}
		if m.viewMode == viewOperation {
			return m, waitForOperationLineCmd(m.logSub, m.operationErrCh)
		}
		return m, waitForLogLineCmd(m.logSub)

	case logStreamEndedMsg:
		m.logProcess = nil
		m.logSub = nil
		return m, nil

	case operationStreamEndedMsg:
		m.operationDone = true
		m.operationErr = msg.err
		m.logProcess = nil
		m.logSub = nil
		m.operationErrCh = nil
		// Reload projects after operation
		cmds := []tea.Cmd{loadProjects, loadRouterStatus, m.spinner.Tick}
		if m.operationReturnView == viewDetail && m.detail != nil {
			cmds = append(cmds, loadDetailCmd(m.detail.AppRoot))
		}
		// Auto-return on success after a short delay; stay on error so user can read output
		if msg.err == nil {
			cmds = append(cmds, scheduleOperationAutoReturn())
		}
		return m, tea.Batch(cmds...)

	case operationAutoReturnMsg:
		// Only act if still on the operation view and the operation succeeded
		if m.viewMode == viewOperation && m.operationDone && m.operationErr == nil {
			returnView := m.operationReturnView
			m.logLines = nil
			m.operationDone = false
			m.operationName = ""
			m.statusMsg = "Operation completed"
			m.viewMode = returnView
		}
		return m, nil

	case operationFinishedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = "Operation completed"
		}
		// Reload projects after an operation
		m.loading = true
		return m, tea.Batch(loadProjects, loadRouterStatus, m.spinner.Tick)

	case operationDetailFinishedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = "Operation completed"
		}
		// Reload both detail and project list after an operation
		m.loading = true
		if m.detail != nil {
			m.detailLoading = true
			return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), loadProjects, m.spinner.Tick)
		}
		return m, tea.Batch(loadProjects, m.spinner.Tick)

	case tickMsg:
		// Auto-refresh based on current view
		switch m.viewMode {
		case viewDetail:
			if m.detail != nil {
				return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), loadRouterStatus, tickCmd())
			}
			return m, tea.Batch(loadRouterStatus, tickCmd())
		case viewLogs, viewOperation:
			// No auto-refresh while streaming logs or operations
			return m, tickCmd()
		case viewDashboard:
			return m, tea.Batch(loadProjects, loadRouterStatus, tickCmd())
		default:
			return m, tickCmd()
		}

	case spinner.TickMsg:
		if m.isLoading() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.viewMode {
		case viewDetail:
			return m.handleDetailKey(msg)
		case viewLogs:
			return m.handleLogKey(msg)
		case viewOperation:
			return m.handleOperationKey(msg)
		default:
			return m.handleDashboardKey(msg)
		}
	}

	return m, nil
}

// isLoading returns true if any loading state is active.
func (m AppModel) isLoading() bool {
	return m.loading || m.detailLoading || (m.viewMode == viewOperation && !m.operationDone)
}

func (m AppModel) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Confirmation overlay takes priority
	if m.confirming {
		if key.Matches(msg, m.keys.Confirm) {
			action := m.confirmAction
			m.confirming = false
			m.confirmAction = ""
			switch action {
			case "start-all":
				m = m.enterOperationView("Starting all projects", viewDashboard)
				return m, startOperationStreamCmd("", "start", "--all")
			case "stop-all":
				m = m.enterOperationView("Stopping all projects", viewDashboard)
				return m, startOperationStreamCmd("", "stop", "--all")
			case "poweroff":
				m = m.enterOperationView("Powering off DDEV", viewDashboard)
				return m, startOperationStreamCmd("", "poweroff")
			}
		}
		// Any other key cancels
		m.confirming = false
		m.confirmAction = ""
		m.statusMsg = ""
		return m, nil
	}

	// If filtering, handle text input
	if m.filtering {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.filtering = false
			m.filterText = ""
			m.cursor = 0
			m.updateDashboardViewport()
			m.dashboardViewport.GotoTop()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			m.filtering = false
			m.updateDashboardViewport()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.cursor = 0
				m.updateDashboardViewport()
				m.dashboardViewport.GotoTop()
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.filterText += msg.String()
				m.cursor = 0
				m.updateDashboardViewport()
				m.dashboardViewport.GotoTop()
			}
			return m, nil
		}
	}

	// Help overlay takes priority
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		if m.filterText != "" {
			m.filterText = ""
			m.cursor = 0
			m.updateDashboardViewport()
			m.dashboardViewport.GotoTop()
			return m, nil
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.updateDashboardViewport()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		filtered := m.filteredProjects()
		if m.cursor < len(filtered)-1 {
			m.cursor++
			m.updateDashboardViewport()
		}
		return m, nil

	case key.Matches(msg, m.keys.PageUp):
		pageSize := m.dashboardViewport.Height
		if pageSize < 1 {
			pageSize = 10
		}
		m.cursor = max(0, m.cursor-pageSize)
		m.updateDashboardViewport()
		return m, nil

	case key.Matches(msg, m.keys.PageDown):
		filtered := m.filteredProjects()
		pageSize := m.dashboardViewport.Height
		if pageSize < 1 {
			pageSize = 10
		}
		m.cursor = min(m.cursor+pageSize, len(filtered)-1)
		m.updateDashboardViewport()
		return m, nil

	case key.Matches(msg, m.keys.Detail):
		if p := m.selectedProject(); p != nil {
			m.viewMode = viewDetail
			m.detail = nil
			m.detailLoading = true
			m.detailViewport.GotoTop()
			m.statusMsg = ""
			return m, tea.Batch(loadDetailCmd(p.AppRoot), m.spinner.Tick)
		}
		return m, nil

	case key.Matches(msg, m.keys.Start):
		if p := m.selectedProject(); p != nil {
			m = m.enterOperationView(fmt.Sprintf("Starting %s", p.Name), viewDashboard)
			return m, startOperationStreamCmd("", "start", p.Name)
		}

	case key.Matches(msg, m.keys.Stop):
		if p := m.selectedProject(); p != nil {
			m = m.enterOperationView(fmt.Sprintf("Stopping %s", p.Name), viewDashboard)
			return m, startOperationStreamCmd("", "stop", p.Name)
		}

	case key.Matches(msg, m.keys.Restart):
		if p := m.selectedProject(); p != nil {
			m = m.enterOperationView(fmt.Sprintf("Restarting %s", p.Name), viewDashboard)
			return m, startOperationStreamCmd("", "restart", p.Name)
		}

	case key.Matches(msg, m.keys.Launch):
		if p := m.selectedProject(); p != nil && p.URL != "" {
			return m, ddevExecCommandInDir(p.AppRoot, "launch")
		}

	case key.Matches(msg, m.keys.Mailpit):
		if p := m.selectedProject(); p != nil && p.URL != "" {
			return m, ddevExecCommandInDir(p.AppRoot, "launch", "-m")
		}

	case key.Matches(msg, m.keys.XHGui):
		if p := m.selectedProject(); p != nil {
			return m, ddevExecCommandInDir(p.AppRoot, "xhgui")
		}

	case key.Matches(msg, m.keys.Poweroff):
		m.confirming = true
		m.confirmAction = "poweroff"
		m.statusMsg = "Poweroff all DDEV projects and containers? (y to confirm, any key to cancel)"
		return m, nil

	case key.Matches(msg, m.keys.StartAll):
		if len(m.projects) > 0 {
			m.confirming = true
			m.confirmAction = "start-all"
			m.statusMsg = fmt.Sprintf("Start all %d projects? (y to confirm, any key to cancel)", len(m.projects))
			return m, nil
		}

	case key.Matches(msg, m.keys.StopAll):
		if len(m.projects) > 0 {
			m.confirming = true
			m.confirmAction = "stop-all"
			m.statusMsg = fmt.Sprintf("Stop all %d projects? (y to confirm, any key to cancel)", len(m.projects))
			return m, nil
		}

	case key.Matches(msg, m.keys.Config):
		return m, ddevConfigCommand()

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.statusMsg = "Refreshing..."
		return m, tea.Batch(loadProjects, loadRouterStatus, m.spinner.Tick)

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterText = ""
		m.cursor = 0
		m.updateDashboardViewport()
		m.dashboardViewport.GotoTop()
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.PageUp), key.Matches(msg, m.keys.Up):
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd

	case key.Matches(msg, m.keys.PageDown), key.Matches(msg, m.keys.Down):
		m.detailViewport, cmd = m.detailViewport.Update(msg)
		return m, cmd

	case key.Matches(msg, m.keys.Back):
		m.viewMode = viewDashboard
		m.detail = nil
		m.statusMsg = ""
		// Update dashboard viewport when returning
		m.updateDashboardViewport()
		return m, nil

	case key.Matches(msg, m.keys.Logs):
		if m.detail != nil {
			m.viewMode = viewLogs
			m.logLines = nil
			return m, startLogStreamCmd(m.detail.AppRoot)
		}
		return m, nil

	case key.Matches(msg, m.keys.Start):
		if m.detail != nil {
			m = m.enterOperationView(fmt.Sprintf("Starting %s", m.detail.Name), viewDetail)
			return m, startOperationStreamCmd(m.detail.AppRoot, "start")
		}

	case key.Matches(msg, m.keys.Stop):
		if m.detail != nil {
			m = m.enterOperationView(fmt.Sprintf("Stopping %s", m.detail.Name), viewDetail)
			return m, startOperationStreamCmd(m.detail.AppRoot, "stop")
		}

	case key.Matches(msg, m.keys.Restart):
		if m.detail != nil {
			m = m.enterOperationView(fmt.Sprintf("Restarting %s", m.detail.Name), viewDetail)
			return m, startOperationStreamCmd(m.detail.AppRoot, "restart")
		}

	case key.Matches(msg, m.keys.Launch):
		if m.detail != nil && len(m.detail.URLs) > 0 {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "launch")
		}

	case key.Matches(msg, m.keys.Mailpit):
		if m.detail != nil && m.detail.MailpitURL != "" {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "launch", "-m")
		}

	case key.Matches(msg, m.keys.XHGui):
		if m.detail != nil {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "xhgui")
		}

	case key.Matches(msg, m.keys.SSH):
		if m.detail != nil {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "ssh")
		}

	case key.Matches(msg, m.keys.Xdebug):
		if m.detail != nil && m.detail.Status == ddevapp.SiteRunning {
			m.statusMsg = "Toggling xdebug..."
			return m, xdebugToggleCmd(m.detail.AppRoot)
		}

	case key.Matches(msg, m.keys.CopyURL):
		if m.detail != nil && len(m.detail.URLs) > 0 {
			m.statusMsg = "Copying URL..."
			return m, copyToClipboard(m.detail.URLs[0])
		}

	case key.Matches(msg, m.keys.Refresh):
		if m.detail != nil {
			m.detailLoading = true
			m.statusMsg = "Refreshing..."
			return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m AppModel) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.logProcess != nil {
			_ = m.logProcess.Kill()
		}
		m.logProcess = nil
		m.logSub = nil
		m.logLines = nil
		m.viewMode = viewDetail
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		if m.logProcess != nil {
			_ = m.logProcess.Kill()
		}
		return m, tea.Quit
	}

	return m, nil
}

func (m AppModel) handleOperationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.logProcess != nil {
			_ = m.logProcess.Kill()
		}
		m.logProcess = nil
		m.logSub = nil
		m.logLines = nil
		m.operationErrCh = nil
		m.operationDone = false
		m.operationErr = nil
		m.operationName = ""
		returnView := m.operationReturnView
		m.viewMode = returnView
		// Reload projects on return
		cmds := []tea.Cmd{loadProjects, loadRouterStatus, m.spinner.Tick}
		if returnView == viewDetail && m.detail != nil {
			m.detailLoading = true
			cmds = append(cmds, loadDetailCmd(m.detail.AppRoot))
		}
		return m, tea.Batch(cmds...)

	case key.Matches(msg, m.keys.Quit):
		if m.logProcess != nil {
			_ = m.logProcess.Kill()
		}
		return m, tea.Quit
	}

	return m, nil
}

// View renders the TUI.
func (m AppModel) View() string {
	if m.showHelp {
		return m.helpView()
	}
	switch m.viewMode {
	case viewDetail:
		return m.detailView()
	case viewLogs:
		return m.logView()
	case viewOperation:
		return m.operationView()
	default:
		return m.dashboardView()
	}
}

// updateDashboardViewport rebuilds dashboard content and adjusts scroll to keep cursor visible.
func (m *AppModel) updateDashboardViewport() {
	if !m.viewportReady {
		return
	}

	// Save current scroll position
	oldOffset := m.dashboardViewport.YOffset

	// Rebuild content with current cursor position
	m.dashboardViewport.SetContent(m.buildDashboardContent())

	// Adjust scroll to keep cursor visible
	viewportHeight := m.dashboardViewport.Height
	if viewportHeight < 1 {
		viewportHeight = 10
	}

	// If cursor is below visible area, scroll down
	if m.cursor >= oldOffset+viewportHeight {
		m.dashboardViewport.YOffset = m.cursor - viewportHeight + 1
	} else if m.cursor < oldOffset {
		// If cursor is above visible area, scroll up
		m.dashboardViewport.YOffset = m.cursor
	} else {
		// Cursor is visible, keep current scroll position
		m.dashboardViewport.YOffset = oldOffset
	}
}

// buildDashboardContent builds the scrollable project list for dashboard view.
func (m AppModel) buildDashboardContent() string {
	var b strings.Builder

	filtered := m.filteredProjects()
	if len(filtered) == 0 {
		if len(m.projects) == 0 {
			b.WriteString("  No DDEV projects found.\n")
			b.WriteString("  Press 'C' to run ddev config, or run it manually in a project directory.\n")
		} else {
			b.WriteString("  No projects match the filter.\n")
		}
		return b.String()
	}

	// Calculate name width: fit the longest project name, with limits
	nameWidth := 16
	for _, fp := range filtered {
		if len(fp.Name) > nameWidth {
			nameWidth = len(fp.Name)
		}
	}
	// Cap: leave room for cursor(2) + status(12) + type(12) + spacing(7) + URL
	maxNameWidth := 30
	if m.width > 0 {
		maxNameWidth = max(16, m.width/3)
	}
	if nameWidth > maxNameWidth {
		nameWidth = maxNameWidth
	}
	typeWidth := 12
	narrow := m.width > 0 && m.width < 60
	if narrow {
		nameWidth = min(nameWidth, max(8, m.width/4))
	}

	for i, p := range filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = m.styles.Cursor.Render("> ")
		}

		displayName := truncate(p.Name, nameWidth)
		name := m.styles.ProjectName.Render(fmt.Sprintf("%-*s", nameWidth, displayName))
		status := m.renderStatus(p.Status)
		pType := m.styles.ProjectType.Render(fmt.Sprintf("%-*s", typeWidth, p.Type))

		url := ""
		if !narrow && p.URL != "" && p.Status == ddevapp.SiteRunning {
			// Truncate URL if it would overflow
			maxURL := m.width - nameWidth - 10 - typeWidth - 8
			if m.width > 0 && maxURL > 10 {
				url = m.styles.URL.Render(truncate(p.URL, maxURL))
			} else if m.width <= 0 {
				url = m.styles.URL.Render(p.URL)
			}
		}

		fmt.Fprintf(&b, "%s%s %s  %s  %s\n", cursor, name, status, pType, url)
	}

	return b.String()
}

func (m AppModel) dashboardView() string {
	var b strings.Builder

	// Title bar
	title := m.styles.Title.Render("DDEV Projects")
	version := m.styles.StatusBar.Render(versionconstants.DdevVersion)
	gap := ""
	if m.width > 0 {
		titleLen := len("DDEV Projects")
		versionLen := len(versionconstants.DdevVersion)
		spaces := m.width - titleLen - versionLen
		if spaces > 0 {
			gap = strings.Repeat(" ", spaces)
		}
	}
	b.WriteString(title + gap + version + "\n")

	// Divider
	dividerWidth := m.width
	if dividerWidth <= 0 {
		dividerWidth = 60
	}
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Filter indicator
	if m.filtering {
		fmt.Fprintf(&b, "Filter: %s█\n\n", m.filterText)
	} else if m.filterText != "" {
		fmt.Fprintf(&b, "Filter: %s (press / to edit, esc to clear)\n\n", m.filterText)
	}

	// Loading/error indicator
	if m.loading && len(m.projects) == 0 {
		fmt.Fprintf(&b, "  %s Loading projects...\n", m.spinner.View())
	} else if m.err != nil && len(m.projects) == 0 {
		fmt.Fprintf(&b, "  Error: %v\n", m.err)
		b.WriteString("  Is Docker running? Press R to retry.\n")
	} else {
		// Use viewport for scrollable project list
		b.WriteString(m.dashboardViewport.View())
	}

	b.WriteString("\n\n")

	// Status message (confirmations, errors) OR hint when all stopped
	// These appear in the same place - status messages take priority
	if m.statusMsg != "" {
		msg := m.statusMsg
		if m.width > 0 {
			msg = ansi.Wrap(msg, m.width, "")
		}
		b.WriteString(msg + "\n")
	} else if len(m.projects) > 0 && m.allStopped() {
		msg := fmt.Sprintf("All projects stopped. Press '%s' to start selected, '%s' to start all.",
			m.keys.Start.Help().Key, m.keys.StartAll.Help().Key)
		if m.width > 0 {
			msg = ansi.Wrap(msg, m.width, "")
		}
		b.WriteString(m.styles.StatusBar.Render(msg) + "\n")
	}

	// Divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Router status
	if m.routerStatus != "" {
		routerLabel := "Router: "
		var rendered string
		switch m.routerStatus {
		case ddevapp.SiteRunning, "healthy":
			rendered = routerLabel + m.styles.Running.Render(m.routerStatus)
		case ddevapp.SiteStopped:
			rendered = routerLabel + m.styles.Stopped.Render(m.routerStatus)
		default:
			rendered = routerLabel + m.styles.Paused.Render(m.routerStatus)
		}
		b.WriteString(rendered + "\n")
	}

	// Divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Key hints (always at bottom)
	b.WriteString(m.dashboardKeyHints())

	return b.String()
}

// buildDetailContent builds the scrollable content for detail view.
func (m AppModel) buildDetailContent() string {
	if m.detail == nil {
		return ""
	}

	d := m.detail
	var content strings.Builder

	// Info grid
	label := func(l string) string { return m.styles.DetailLabel.Render(l) }
	val := func(v string) string { return m.styles.DetailValue.Render(v) }

	dbStr := d.DatabaseType
	if d.DatabaseVersion != "" {
		dbStr += ":" + d.DatabaseVersion
	}

	xdebugStr := "off"
	if d.XdebugEnabled {
		xdebugStr = "on"
	}

	perfStr := d.PerformanceMode
	if perfStr == "" {
		perfStr = "none"
	}

	fmt.Fprintf(&content, " %s %s    %s %s\n", label("Type:"), val(fmt.Sprintf("%-14s", d.Type)), label("PHP:"), val(d.PHPVersion))
	fmt.Fprintf(&content, " %s %s    %s %s\n", label("Webserver:"), val(fmt.Sprintf("%-14s", d.WebserverType)), label("Node.js:"), val(d.NodeJSVersion))
	fmt.Fprintf(&content, " %s %s    %s %s\n", label("Docroot:"), val(fmt.Sprintf("%-14s", d.Docroot)), label("Perf:"), val(perfStr))
	fmt.Fprintf(&content, " %s %s    %s %s\n", label("Database:"), val(fmt.Sprintf("%-14s", dbStr)), label("Xdebug:"), val(xdebugStr))
	content.WriteString("\n")

	// URLs
	if len(d.URLs) > 0 {
		content.WriteString(" " + label("URLs:") + "\n")
		// Calculate max URL width (leave room for indent and ellipsis)
		maxURLWidth := m.width - 5 // 3 spaces indent + margin
		if maxURLWidth < 20 {
			maxURLWidth = 60 // Default for unknown width
		}
		for _, u := range d.URLs {
			displayURL := truncate(u, maxURLWidth)
			content.WriteString("   " + m.styles.URL.Render(displayURL) + "\n")
		}
		content.WriteString("\n")
	}

	// Mailpit + DB port
	if d.MailpitURL != "" {
		maxURLWidth := m.width - 12 // "Mailpit: " length
		if maxURLWidth < 20 {
			maxURLWidth = 60
		}
		displayMailpit := truncate(d.MailpitURL, maxURLWidth)
		fmt.Fprintf(&content, " %s %s\n", label("Mailpit:"), m.styles.URL.Render(displayMailpit))
	}
	if d.DBPublishedPort != "" {
		fmt.Fprintf(&content, " %s %s\n", label("DB Port:"), val(d.DBPublishedPort))
	}

	// Add-ons
	if len(d.Addons) > 0 {
		addonsStr := strings.Join(d.Addons, ", ")
		// Wrap add-ons if too long
		maxAddonsWidth := m.width - 12 // "Add-ons: " length
		if maxAddonsWidth < 20 {
			maxAddonsWidth = 60
		}
		addonsStr = ansi.Wrap(addonsStr, maxAddonsWidth, "")
		// Add proper indentation for wrapped lines
		lines := strings.Split(addonsStr, "\n")
		fmt.Fprintf(&content, " %s %s\n", label("Add-ons:"), val(lines[0]))
		for i := 1; i < len(lines); i++ {
			fmt.Fprintf(&content, "           %s\n", val(lines[i]))
		}
	}

	content.WriteString("\n")

	// Services (sorted: web first, db second, rest alphabetical)
	if len(d.Services) > 0 {
		sorted := sortServices(d.Services)
		content.WriteString(" " + label("Services:") + "\n")

		// Build list of services with compact status (no padding)
		maxLineWidth := m.width - 3 // indent
		if maxLineWidth < 20 {
			maxLineWidth = 60
		}

		var currentLine strings.Builder
		currentLine.WriteString("   ")
		lineLen := 0

		for i, svc := range sorted {
			// Render service without padded status
			var statusRendered string
			switch svc.Status {
			case ddevapp.SiteRunning:
				statusRendered = m.styles.Running.Render(svc.Status)
			case ddevapp.SiteStopped:
				statusRendered = m.styles.Stopped.Render(svc.Status)
			case ddevapp.SitePaused:
				statusRendered = m.styles.Paused.Render(svc.Status)
			default:
				statusRendered = m.styles.Stopped.Render(svc.Status)
			}

			svcText := fmt.Sprintf("%s %s", m.styles.ProjectName.Render(svc.Name), statusRendered)
			svcLen := len(svc.Name) + 1 + len(svc.Status)

			// Add comma and space for all but first item on a line
			if lineLen > 0 {
				svcText = ", " + svcText
				svcLen += 2
			}

			// Check if adding this service would exceed line width
			if lineLen > 0 && lineLen+svcLen > maxLineWidth {
				// Add comma before line break if not last service
				if i < len(sorted)-1 {
					currentLine.WriteString(",")
				}
				// Start new line
				content.WriteString(currentLine.String() + "\n")
				currentLine.Reset()
				currentLine.WriteString("   ")
				// Don't add comma at start of new line
				svcText = fmt.Sprintf("%s %s", m.styles.ProjectName.Render(svc.Name), statusRendered)
				lineLen = len(svc.Name) + 1 + len(svc.Status)
			} else {
				lineLen += svcLen
			}

			currentLine.WriteString(svcText)
		}

		// Write final line
		if currentLine.Len() > 3 {
			content.WriteString(currentLine.String() + "\n")
		}
	}

	// Status message
	if m.statusMsg != "" {
		content.WriteString("\n" + m.statusMsg)
	}

	return content.String()
}

func (m AppModel) detailView() string {
	var b strings.Builder

	dividerWidth := m.width
	if dividerWidth <= 0 {
		dividerWidth = 60
	}

	if m.detailLoading || m.detail == nil {
		b.WriteString(m.styles.Title.Render("DDEV Project") + "\n")
		b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")
		fmt.Fprintf(&b, "\n  %s Loading project detail...\n", m.spinner.View())
		return b.String()
	}

	d := m.detail

	// Title bar: project name + status
	titleText := fmt.Sprintf("DDEV Project: %s", d.Name)
	title := m.styles.Title.Render(titleText)
	status := m.renderStatus(d.Status)
	gap := ""
	if m.width > 0 {
		spaces := m.width - len(titleText) - len(d.Status)
		if spaces > 0 {
			gap = strings.Repeat(" ", spaces)
		}
	}
	b.WriteString(title + gap + status + "\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")
	b.WriteString("\n")

	// Use viewport for scrollable content
	b.WriteString(m.detailViewport.View())

	// Bottom divider
	b.WriteString("\n" + m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Key hints
	b.WriteString(m.detailKeyHints())

	return b.String()
}

func (m AppModel) logView() string {
	var b strings.Builder

	dividerWidth := m.width
	if dividerWidth <= 0 {
		dividerWidth = 60
	}

	name := ""
	if m.detail != nil {
		name = m.detail.Name
	}

	// Title
	titleText := fmt.Sprintf("DDEV Logs: %s", name)
	title := m.styles.Title.Render(titleText)
	b.WriteString(title + "\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	if len(m.logLines) == 0 {
		fmt.Fprintf(&b, "\n  %s Waiting for log output...\n", m.spinner.View())
	} else {
		// Show the last N lines that fit the terminal (auto-scroll to bottom)
		viewHeight := m.height - 4 // title, divider, bottom divider, hints
		if viewHeight < 5 {
			viewHeight = 20
		}
		start := 0
		if len(m.logLines) > viewHeight {
			start = len(m.logLines) - viewHeight
		}
		for _, line := range m.logLines[start:] {
			b.WriteString(line + "\n")
		}
	}

	// Bottom divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")
	b.WriteString(m.logKeyHints())

	return b.String()
}

func (m AppModel) logKeyHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"esc", "back"},
		{"q", "quit"},
	}
	return m.renderHints(hints)
}

func (m AppModel) operationView() string {
	var b strings.Builder

	dividerWidth := m.width
	if dividerWidth <= 0 {
		dividerWidth = 60
	}

	// Title
	titleText := m.operationName
	if titleText == "" {
		titleText = "Operation"
	}
	title := m.styles.Title.Render(titleText)
	b.WriteString(title + "\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	if len(m.logLines) == 0 && !m.operationDone {
		fmt.Fprintf(&b, "\n  %s Running...\n", m.spinner.View())
	} else {
		// Show the last N lines that fit the terminal (auto-scroll to bottom)
		viewHeight := m.height - 6 // title, divider, status line, bottom divider, hints, margin
		if viewHeight < 5 {
			viewHeight = 20
		}
		start := 0
		if len(m.logLines) > viewHeight {
			start = len(m.logLines) - viewHeight
		}
		for _, line := range m.logLines[start:] {
			b.WriteString(line + "\n")
		}
	}

	// Status line
	if m.operationDone {
		if m.operationErr != nil {
			b.WriteString(m.styles.Stopped.Render(fmt.Sprintf("Failed: %v", m.operationErr)) + "\n")
		} else {
			b.WriteString(m.styles.Running.Render("Completed — returning shortly...") + "\n")
		}
	}

	// Bottom divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")
	b.WriteString(m.operationKeyHints())

	return b.String()
}

func (m AppModel) operationKeyHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"esc", "back"},
		{"q", "quit"},
	}
	return m.renderHints(hints)
}

// sortServices returns services sorted: web first, db second, rest alphabetical.
func sortServices(services []ServiceInfo) []ServiceInfo {
	sorted := make([]ServiceInfo, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool {
		return serviceOrder(sorted[i].Name) < serviceOrder(sorted[j].Name)
	})
	return sorted
}

// serviceOrder returns a sort key: web=0, db=1, everything else=2+name.
func serviceOrder(name string) string {
	switch name {
	case "web":
		return "0"
	case "db":
		return "1"
	default:
		return "2" + name
	}
}

// truncate shortens a string to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (m AppModel) renderStatus(status string) string {
	padded := fmt.Sprintf("%-10s", status)
	switch status {
	case ddevapp.SiteRunning:
		return m.styles.Running.Render(padded)
	case ddevapp.SiteStopped:
		return m.styles.Stopped.Render(padded)
	case ddevapp.SitePaused:
		return m.styles.Paused.Render(padded)
	default:
		return m.styles.Stopped.Render(padded)
	}
}

func (m AppModel) dashboardKeyHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"s", "start"},
		{"S", "stop"},
		{"r", "restart"},
		{"a", "start all"},
		{"A", "stop all"},
		{"P", "poweroff"},
		{"l", "launch"},
		{"m", "mailpit"},
		{"x", "xhgui"},
		{"C", "config"},
		{"enter", "detail"},
		{"/", "filter"},
		{"?", "help"},
		{"q", "quit"},
	}
	return m.renderHints(hints)
}

func (m AppModel) detailKeyHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"s", "start"},
		{"S", "stop"},
		{"r", "restart"},
		{"l", "launch"},
		{"m", "mailpit"},
		{"x", "xhgui"},
		{"X", "xdebug"},
		{"c", "copy url"},
		{"e", "ssh"},
		{"L", "logs"},
		{"R", "refresh"},
		{"esc", "back"},
	}
	return m.renderHints(hints)
}

func (m AppModel) renderHints(hints []struct {
	key  string
	desc string
}) string {
	const sep = "  "
	const sepLen = 2

	var lines []string
	var currentParts []string
	currentLen := 0

	for _, h := range hints {
		rendered := m.styles.HelpKey.Render(h.key) + " " + m.styles.HelpDesc.Render(h.desc)
		hintLen := len(h.key) + 1 + len(h.desc)

		if len(currentParts) == 0 {
			currentParts = append(currentParts, rendered)
			currentLen = hintLen
		} else if m.width > 0 && currentLen+sepLen+hintLen > m.width {
			lines = append(lines, strings.Join(currentParts, sep))
			currentParts = []string{rendered}
			currentLen = hintLen
		} else {
			currentParts = append(currentParts, rendered)
			currentLen += sepLen + hintLen
		}
	}
	if len(currentParts) > 0 {
		lines = append(lines, strings.Join(currentParts, sep))
	}
	return strings.Join(lines, "\n")
}

// keyHintsView returns the dashboard key hints (kept for backward compatibility).
func (m AppModel) keyHintsView() string {
	return m.dashboardKeyHints()
}

func (m AppModel) helpView() string {
	help := `
DDEV TUI Dashboard - Help

Navigation:
  up/k, down/j   Move selection
  enter/d         Open project detail
  /               Filter projects
  esc             Back / clear filter

Actions:
  s               Start selected project
  S               Stop selected project
  r               Restart selected project
  a               Start all projects
  A               Stop all projects
  P               Poweroff all DDEV projects and containers
  C               Run ddev config interactively
  l               Launch project URL in browser
  m               Launch Mailpit in browser
  x               Launch XHGui (enable xhprof + open UI)
  X               Toggle Xdebug on/off (from detail view)
  c               Copy primary URL to clipboard (from detail view)
  e               SSH into web container (from detail view)
  L               Follow logs (from detail view)
  R               Refresh

Other:
  ?               Toggle this help
  q, ctrl+c       Quit

Press any key to close this help.
`
	return m.styles.HelpOverlay.Render(help)
}
