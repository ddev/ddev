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

	case operationFinishedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = "Operation completed"
		}
		// Reload projects after an operation
		m.loading = true
		return m, loadProjects

	case tickMsg:
		// Auto-refresh project list
		return m, tea.Batch(loadProjects, tickCmd())

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// View renders the TUI.
func (m AppModel) View() string {
	if m.showHelp {
		return m.helpView()
	}
	return m.dashboardView()
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
	b.WriteString(m.keyHintsView())

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

func (m AppModel) keyHintsView() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"s", "start"},
		{"x", "stop"},
		{"r", "restart"},
		{"o", "open"},
		{"/", "filter"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			m.styles.HelpKey.Render(h.key)+" "+m.styles.HelpDesc.Render(h.desc))
	}
	return strings.Join(parts, "  ")
}

func (m AppModel) helpView() string {
	help := `
DDEV TUI Dashboard - Help

Navigation:
  up/k, down/j   Move selection
  /               Filter projects
  esc             Clear filter / close

Actions:
  s               Start selected project
  x               Stop selected project
  r               Restart selected project
  o               Open project URL in browser
  R               Refresh project list

Other:
  ?               Toggle this help
  q, ctrl+c       Quit

Press any key to close this help.
`
	return m.styles.HelpOverlay.Render(help)
}
