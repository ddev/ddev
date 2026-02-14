package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/versionconstants"
)

const refreshInterval = 5 * time.Second

// View mode constants.
const (
	viewDashboard = iota
	viewDetail
	viewLogs
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

	// Log view
	logContent string
	logService string
	logScroll  int
	logLoading bool
}

// NewAppModel creates a new TUI model.
func NewAppModel() AppModel {
	return AppModel{
		keys:    DefaultKeyMap(),
		styles:  NewStyles(),
		loading: true,
	}
}

// Init starts the initial data load and refresh ticker.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		loadProjects,
		tickCmd(),
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

// selectedProject returns the currently selected project, or nil.
func (m AppModel) selectedProject() *ProjectInfo {
	filtered := m.filteredProjects()
	if len(filtered) == 0 || m.cursor >= len(filtered) {
		return nil
	}
	return &filtered[m.cursor]
}

// Update handles messages and key presses.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case projectsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.projects = msg.projects
		// Keep cursor in bounds
		filtered := m.filteredProjects()
		if m.cursor >= len(filtered) {
			m.cursor = max(0, len(filtered)-1)
		}
		return m, nil

	case projectDetailLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading detail: %v", msg.err)
			m.viewMode = viewDashboard
			return m, nil
		}
		m.detail = &msg.detail
		return m, nil

	case logsLoadedMsg:
		m.logLoading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading logs: %v", msg.err)
			return m, nil
		}
		m.logContent = msg.logs
		m.logService = msg.service
		m.logScroll = 0
		return m, nil

	case operationFinishedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = "Operation completed"
		}
		// Reload projects after an operation
		m.loading = true
		return m, loadProjects

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
			return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), loadProjects)
		}
		return m, loadProjects

	case tickMsg:
		// Auto-refresh based on current view
		switch m.viewMode {
		case viewDetail:
			if m.detail != nil {
				return m, tea.Batch(loadDetailCmd(m.detail.AppRoot), tickCmd())
			}
			return m, tickCmd()
		case viewDashboard:
			return m, tea.Batch(loadProjects, tickCmd())
		default:
			return m, tickCmd()
		}

	case tea.KeyMsg:
		switch m.viewMode {
		case viewDetail:
			return m.handleDetailKey(msg)
		case viewLogs:
			return m.handleLogKey(msg)
		default:
			return m.handleDashboardKey(msg)
		}
	}

	return m, nil
}

func (m AppModel) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If filtering, handle text input
	if m.filtering {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.filtering = false
			m.filterText = ""
			m.cursor = 0
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			m.filtering = false
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.cursor = 0
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.filterText += msg.String()
				m.cursor = 0
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
			return m, nil
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		filtered := m.filteredProjects()
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Detail):
		if p := m.selectedProject(); p != nil {
			m.viewMode = viewDetail
			m.detail = nil
			m.detailLoading = true
			m.statusMsg = ""
			return m, loadDetailCmd(p.AppRoot)
		}
		return m, nil

	case key.Matches(msg, m.keys.Start):
		if p := m.selectedProject(); p != nil {
			m.statusMsg = fmt.Sprintf("Starting %s...", p.Name)
			return m, ddevExecCommand("start", p.Name)
		}

	case key.Matches(msg, m.keys.Stop):
		if p := m.selectedProject(); p != nil {
			m.statusMsg = fmt.Sprintf("Stopping %s...", p.Name)
			return m, ddevExecCommand("stop", p.Name)
		}

	case key.Matches(msg, m.keys.Restart):
		if p := m.selectedProject(); p != nil {
			m.statusMsg = fmt.Sprintf("Restarting %s...", p.Name)
			return m, ddevExecCommand("restart", p.Name)
		}

	case key.Matches(msg, m.keys.Open):
		if p := m.selectedProject(); p != nil && p.URL != "" {
			return m, ddevExecCommandInDir(p.AppRoot, "launch")
		}

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.statusMsg = "Refreshing..."
		return m, loadProjects

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterText = ""
		m.cursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.viewMode = viewDashboard
		m.detail = nil
		m.statusMsg = ""
		return m, nil

	case key.Matches(msg, m.keys.Logs):
		if m.detail != nil {
			m.viewMode = viewLogs
			m.logService = "web"
			m.logContent = ""
			m.logScroll = 0
			m.logLoading = true
			return m, loadLogsCmd(m.detail.AppRoot, "web")
		}
		return m, nil

	case key.Matches(msg, m.keys.Start):
		if m.detail != nil {
			m.statusMsg = fmt.Sprintf("Starting %s...", m.detail.Name)
			return m, ddevExecCommandDetail(m.detail.AppRoot, "start")
		}

	case key.Matches(msg, m.keys.Stop):
		if m.detail != nil {
			m.statusMsg = fmt.Sprintf("Stopping %s...", m.detail.Name)
			return m, ddevExecCommandDetail(m.detail.AppRoot, "stop")
		}

	case key.Matches(msg, m.keys.Restart):
		if m.detail != nil {
			m.statusMsg = fmt.Sprintf("Restarting %s...", m.detail.Name)
			return m, ddevExecCommandDetail(m.detail.AppRoot, "restart")
		}

	case key.Matches(msg, m.keys.Open):
		if m.detail != nil && len(m.detail.URLs) > 0 {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "launch")
		}

	case key.Matches(msg, m.keys.SSH):
		if m.detail != nil {
			return m, ddevExecCommandDetail(m.detail.AppRoot, "ssh")
		}

	case key.Matches(msg, m.keys.Refresh):
		if m.detail != nil {
			m.detailLoading = true
			m.statusMsg = "Refreshing..."
			return m, loadDetailCmd(m.detail.AppRoot)
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m AppModel) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.viewMode = viewDetail
		m.logContent = ""
		m.logScroll = 0
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.logScroll > 0 {
			m.logScroll--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		lines := strings.Split(m.logContent, "\n")
		viewHeight := m.logViewHeight()
		maxScroll := len(lines) - viewHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.logScroll < maxScroll {
			m.logScroll++
		}
		return m, nil

	case key.Matches(msg, m.keys.LogWeb):
		if m.detail != nil && m.logService != "web" {
			m.logService = "web"
			m.logContent = ""
			m.logScroll = 0
			m.logLoading = true
			return m, loadLogsCmd(m.detail.AppRoot, "web")
		}
		return m, nil

	case key.Matches(msg, m.keys.LogDB):
		if m.detail != nil && m.logService != "db" {
			m.logService = "db"
			m.logContent = ""
			m.logScroll = 0
			m.logLoading = true
			return m, loadLogsCmd(m.detail.AppRoot, "db")
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		if m.detail != nil {
			m.logLoading = true
			return m, loadLogsCmd(m.detail.AppRoot, m.logService)
		}

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

// logViewHeight returns the number of lines available for log content.
func (m AppModel) logViewHeight() int {
	// Reserve lines for: title, divider, bottom divider, key hints
	overhead := 4
	h := m.height - overhead
	if h < 5 {
		h = 20
	}
	return h
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
	default:
		return m.dashboardView()
	}
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
		b.WriteString(fmt.Sprintf("Filter: %s█\n\n", m.filterText))
	} else if m.filterText != "" {
		b.WriteString(fmt.Sprintf("Filter: %s (press / to edit, esc to clear)\n\n", m.filterText))
	}

	// Loading indicator
	if m.loading && len(m.projects) == 0 {
		b.WriteString("  Loading projects...\n")
	} else {
		filtered := m.filteredProjects()
		if len(filtered) == 0 {
			if len(m.projects) == 0 {
				b.WriteString("  No DDEV projects found.\n")
				b.WriteString("  Run 'ddev config' in a project directory to get started.\n")
			} else {
				b.WriteString("  No projects match the filter.\n")
			}
		} else {
			for i, p := range filtered {
				cursor := "  "
				if i == m.cursor {
					cursor = m.styles.Cursor.Render("> ")
				}

				name := m.styles.ProjectName.Render(fmt.Sprintf("%-16s", p.Name))
				status := m.renderStatus(p.Status)
				pType := m.styles.ProjectType.Render(fmt.Sprintf("%-12s", p.Type))

				url := ""
				if p.URL != "" && p.Status == ddevapp.SiteRunning {
					url = m.styles.URL.Render(p.URL)
				}

				b.WriteString(fmt.Sprintf("%s%s %s  %s  %s\n", cursor, name, status, pType, url))
			}
		}
	}

	b.WriteString("\n")

	// Router status
	if m.routerStatus != "" {
		b.WriteString(fmt.Sprintf("Router: %s\n", m.routerStatus))
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString(m.statusMsg + "\n")
	}

	// Bottom divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	// Key hints
	b.WriteString(m.dashboardKeyHints())

	return b.String()
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
		b.WriteString("\n  Loading project detail...\n")
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

	b.WriteString(fmt.Sprintf(" %s %s    %s %s\n", label("Type:"), val(fmt.Sprintf("%-14s", d.Type)), label("PHP:"), val(d.PHPVersion)))
	b.WriteString(fmt.Sprintf(" %s %s    %s %s\n", label("Webserver:"), val(fmt.Sprintf("%-14s", d.WebserverType)), label("Node.js:"), val(d.NodeJSVersion)))
	b.WriteString(fmt.Sprintf(" %s %s    %s %s\n", label("Docroot:"), val(fmt.Sprintf("%-14s", d.Docroot)), label("Perf:"), val(perfStr)))
	b.WriteString(fmt.Sprintf(" %s %s    %s %s\n", label("Database:"), val(fmt.Sprintf("%-14s", dbStr)), label("Xdebug:"), val(xdebugStr)))
	b.WriteString("\n")

	// URLs
	if len(d.URLs) > 0 {
		b.WriteString(" " + label("URLs:") + "\n")
		for _, u := range d.URLs {
			b.WriteString("   " + m.styles.URL.Render(u) + "\n")
		}
		b.WriteString("\n")
	}

	// Mailpit + DB port
	if d.MailpitURL != "" {
		b.WriteString(fmt.Sprintf(" %s %s\n", label("Mailpit:"), m.styles.URL.Render(d.MailpitURL)))
	}
	if d.DBPublishedPort != "" {
		b.WriteString(fmt.Sprintf(" %s %s\n", label("DB Port:"), val(d.DBPublishedPort)))
	}

	// Add-ons
	if len(d.Addons) > 0 {
		b.WriteString(fmt.Sprintf(" %s %s\n", label("Add-ons:"), val(strings.Join(d.Addons, ", "))))
	}

	b.WriteString("\n")

	// Services
	if len(d.Services) > 0 {
		b.WriteString(" " + label("Services:") + "\n   ")
		var svcParts []string
		for _, svc := range d.Services {
			svcParts = append(svcParts, fmt.Sprintf("%s %s", m.styles.ProjectName.Render(svc.Name), m.renderStatus(svc.Status)))
		}
		b.WriteString(strings.Join(svcParts, "  "))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Status message
	if m.statusMsg != "" {
		b.WriteString(m.statusMsg + "\n")
	}

	// Bottom divider
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

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
	titleText := fmt.Sprintf("DDEV Logs: %s (%s)", name, m.logService)
	title := m.styles.Title.Render(titleText)
	b.WriteString(title + "\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")

	if m.logLoading {
		b.WriteString("\n  Loading logs...\n")
	} else if m.logContent == "" {
		b.WriteString("\n  No log output.\n")
	} else {
		lines := strings.Split(m.logContent, "\n")
		viewHeight := m.logViewHeight()

		start := m.logScroll
		if start > len(lines) {
			start = len(lines)
		}
		end := start + viewHeight
		if end > len(lines) {
			end = len(lines)
		}

		visible := lines[start:end]
		b.WriteString("\n")
		for _, line := range visible {
			b.WriteString(line + "\n")
		}
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString(m.statusMsg + "\n")
	}

	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", dividerWidth)) + "\n")
	b.WriteString(m.logKeyHints())

	return b.String()
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
		{"x", "stop"},
		{"r", "restart"},
		{"o", "open"},
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
		{"x", "stop"},
		{"r", "restart"},
		{"o", "open"},
		{"e", "ssh"},
		{"l", "logs"},
		{"R", "refresh"},
		{"esc", "back"},
	}
	return m.renderHints(hints)
}

func (m AppModel) logKeyHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"w", "web"},
		{"d", "db"},
		{"j/k", "scroll"},
		{"R", "refresh"},
		{"esc", "back"},
	}
	return m.renderHints(hints)
}

func (m AppModel) renderHints(hints []struct {
	key  string
	desc string
}) string {
	var parts []string
	for _, h := range hints {
		parts = append(parts,
			m.styles.HelpKey.Render(h.key)+" "+m.styles.HelpDesc.Render(h.desc))
	}
	return strings.Join(parts, "  ")
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
  x               Stop selected project
  r               Restart selected project
  o               Open project URL in browser
  e               SSH into web container (from detail view)
  l               View logs (from detail view)
  R               Refresh

Other:
  ?               Toggle this help
  q, ctrl+c       Quit

Press any key to close this help.
`
	return m.styles.HelpOverlay.Render(help)
}
