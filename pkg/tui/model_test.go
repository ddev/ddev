package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

func TestNewAppModel(t *testing.T) {
	m := NewAppModel()
	require.True(t, m.loading, "model should start in loading state")
	require.Empty(t, m.projects, "model should start with no projects")
	require.Equal(t, 0, m.cursor)
	require.False(t, m.showHelp)
	require.False(t, m.filtering)
}

func TestProjectsLoadedMsg(t *testing.T) {
	m := NewAppModel()
	m.loading = true

	projects := []ProjectInfo{
		{Name: "site-a", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://site-a.ddev.site"},
		{Name: "site-b", Status: ddevapp.SiteStopped, Type: "wordpress"},
	}

	updated, _ := m.Update(projectsLoadedMsg{projects: projects})
	model := updated.(AppModel)

	require.False(t, model.loading, "loading should be false after projects loaded")
	require.Len(t, model.projects, 2)
	require.Equal(t, "site-a", model.projects[0].Name)
	require.Equal(t, "site-b", model.projects[1].Name)
}

func TestProjectsLoadedError(t *testing.T) {
	m := NewAppModel()
	m.loading = true

	updated, _ := m.Update(projectsLoadedMsg{err: errTest})
	model := updated.(AppModel)

	require.False(t, model.loading)
	require.Error(t, model.err)
}

var errTest = errTestType{}

type errTestType struct{}

func (errTestType) Error() string { return "test error" }

func TestCursorNavigation(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(AppModel)
	require.Equal(t, 1, model.cursor)

	// Move down again
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(AppModel)
	require.Equal(t, 2, model.cursor)

	// Move down at bottom - should stay
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(AppModel)
	require.Equal(t, 2, model.cursor)

	// Move up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 1, model.cursor)

	// Move up again
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 0, model.cursor)

	// Move up at top - should stay
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 0, model.cursor)
}

func TestFilteredProjects(t *testing.T) {
	m := NewAppModel()
	m.projects = []ProjectInfo{
		{Name: "drupal-site", Type: "drupal", Status: ddevapp.SiteRunning},
		{Name: "wp-blog", Type: "wordpress", Status: ddevapp.SiteStopped},
		{Name: "laravel-app", Type: "laravel", Status: ddevapp.SiteRunning},
	}

	// No filter returns all
	require.Len(t, m.filteredProjects(), 3)

	// Filter by name
	m.filterText = "drupal"
	filtered := m.filteredProjects()
	require.Len(t, filtered, 1)
	require.Equal(t, "drupal-site", filtered[0].Name)

	// Filter by type
	m.filterText = "wordpress"
	filtered = m.filteredProjects()
	require.Len(t, filtered, 1)
	require.Equal(t, "wp-blog", filtered[0].Name)

	// Filter by status
	m.filterText = "running"
	filtered = m.filteredProjects()
	require.Len(t, filtered, 2)

	// No matches
	m.filterText = "nonexistent"
	filtered = m.filteredProjects()
	require.Empty(t, filtered)

	// Case insensitive
	m.filterText = "DRUPAL"
	filtered = m.filteredProjects()
	require.Len(t, filtered, 1)
}

func TestSelectedProject(t *testing.T) {
	m := NewAppModel()

	// No projects
	require.Nil(t, m.selectedProject())

	m.projects = []ProjectInfo{
		{Name: "a"}, {Name: "b"},
	}

	// Cursor at 0
	p := m.selectedProject()
	require.NotNil(t, p)
	require.Equal(t, "a", p.Name)

	// Cursor at 1
	m.cursor = 1
	p = m.selectedProject()
	require.NotNil(t, p)
	require.Equal(t, "b", p.Name)

	// Cursor out of bounds
	m.cursor = 5
	require.Nil(t, m.selectedProject())
}

func TestFilterMode(t *testing.T) {
	m := NewAppModel()
	m.projects = []ProjectInfo{{Name: "a"}, {Name: "b"}}

	// Enter filter mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model := updated.(AppModel)
	require.True(t, model.filtering)

	// Type a character
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(AppModel)
	require.Equal(t, "a", model.filterText)

	// Backspace
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updated.(AppModel)
	require.Equal(t, "", model.filterText)

	// Type and press enter to confirm
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = updated.(AppModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)
	require.False(t, model.filtering)
	require.Equal(t, "b", model.filterText)

	// Esc clears filter while in filtering mode
	model.filtering = true
	model.filterText = "test"
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(AppModel)
	require.False(t, model.filtering)
	require.Equal(t, "", model.filterText)

	// Esc clears filter when NOT in filtering mode (after Enter confirmed)
	model.filtering = false
	model.filterText = "active-filter"
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(AppModel)
	require.Equal(t, "", model.filterText, "esc should clear filter even outside filter mode")
	require.Equal(t, 0, model.cursor)
}

func TestHelpToggle(t *testing.T) {
	m := NewAppModel()

	// Show help
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model := updated.(AppModel)
	require.True(t, model.showHelp)

	// Any key closes help
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(AppModel)
	require.False(t, model.showHelp)
}

func TestQuit(t *testing.T) {
	m := NewAppModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd, "quit should return a command")

	// The cmd should produce a QuitMsg
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	require.True(t, ok, "quit command should produce QuitMsg")
}

func TestDashboardView(t *testing.T) {
	m := NewAppModel()
	m.width = 60
	m.height = 24
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site"},
		{Name: "other", Status: ddevapp.SiteStopped, Type: "wordpress"},
	}

	view := m.View()

	require.True(t, strings.Contains(view, "DDEV Projects"), "view should contain title")
	require.True(t, strings.Contains(view, "mysite"), "view should contain project name")
	require.True(t, strings.Contains(view, "other"), "view should contain second project")
	require.True(t, strings.Contains(view, "drupal"), "view should contain project type")
	require.True(t, strings.Contains(view, "start"), "view should contain key hints")
	require.True(t, strings.Contains(view, "quit"), "view should contain quit hint")
}

func TestHelpView(t *testing.T) {
	m := NewAppModel()
	m.showHelp = true

	view := m.View()

	require.True(t, strings.Contains(view, "Help"), "help view should contain Help title")
	require.True(t, strings.Contains(view, "Navigation"), "help view should contain Navigation section")
	require.True(t, strings.Contains(view, "Actions"), "help view should contain Actions section")
}

func TestEmptyProjectsView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 60

	view := m.View()
	require.True(t, strings.Contains(view, "No DDEV projects found"), "should show empty message")
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	require.Equal(t, 120, model.width)
	require.Equal(t, 40, model.height)
}

func TestOperationFinishedMsg(t *testing.T) {
	m := NewAppModel()
	m.projects = []ProjectInfo{{Name: "test"}}

	// Successful operation
	updated, cmd := m.Update(operationFinishedMsg{err: nil})
	model := updated.(AppModel)
	require.Equal(t, "Operation completed", model.statusMsg)
	require.True(t, model.loading, "should reload projects after operation")
	require.NotNil(t, cmd)

	// Failed operation
	updated, _ = m.Update(operationFinishedMsg{err: errTest})
	model = updated.(AppModel)
	require.True(t, strings.Contains(model.statusMsg, "Error"))
}

func TestCursorBoundsOnProjectReload(t *testing.T) {
	m := NewAppModel()
	m.cursor = 5

	// Load fewer projects than cursor position
	updated, _ := m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{{Name: "a"}, {Name: "b"}},
	})
	model := updated.(AppModel)
	require.Equal(t, 1, model.cursor, "cursor should be clamped to last item")
}

func TestRenderStatus(t *testing.T) {
	m := NewAppModel()
	m.styles = NewStyles()

	// Just verify these don't panic
	_ = m.renderStatus(ddevapp.SiteRunning)
	_ = m.renderStatus(ddevapp.SiteStopped)
	_ = m.renderStatus(ddevapp.SitePaused)
	_ = m.renderStatus("unknown")
}

func TestExtractProjectInfo(t *testing.T) {
	// This tests the struct conversion; it doesn't call Docker.
	info := ProjectInfo{
		Name:    "test-project",
		Status:  "running",
		Type:    "php",
		URL:     "https://test-project.ddev.site",
		AppRoot: "/tmp/test",
	}

	require.Equal(t, "test-project", info.Name)
	require.Equal(t, "running", info.Status)
	require.Equal(t, "php", info.Type)
	require.Equal(t, "https://test-project.ddev.site", info.URL)
	require.Equal(t, "/tmp/test", info.AppRoot)
}

// --- Phase 2 tests: Detail View & Log Viewer ---

func sampleDetail() ProjectDetail {
	return ProjectDetail{
		Name:            "mysite",
		Status:          ddevapp.SiteRunning,
		Type:            "drupal",
		PHPVersion:      "8.2",
		WebserverType:   "nginx-fpm",
		NodeJSVersion:   "20",
		Docroot:         "web",
		DatabaseType:    "mariadb",
		DatabaseVersion: "10.11",
		XdebugEnabled:   false,
		PerformanceMode: "mutagen",
		URLs:            []string{"https://mysite.ddev.site"},
		MailpitURL:      "https://mysite.ddev.site:8026",
		DBPublishedPort: "127.0.0.1:32773",
		Addons:          []string{"ddev-redis", "ddev-elasticsearch"},
		Services: []ServiceInfo{
			{Name: "web", Status: ddevapp.SiteRunning},
			{Name: "db", Status: ddevapp.SiteRunning},
		},
		AppRoot: "/tmp/mysite",
	}
}

func TestViewTransitionDashboardToDetail(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", AppRoot: "/tmp/mysite"},
	}

	// Press Enter to open detail
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "should switch to detail view")
	require.True(t, model.detailLoading, "should be loading detail")
	require.Nil(t, model.detail, "detail should be nil while loading")
	require.NotNil(t, cmd, "should return a command to load detail")
}

func TestViewTransitionDashboardToDetailWithD(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", AppRoot: "/tmp/mysite"},
	}

	// Press 'd' to open detail
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "'d' should switch to detail view")
	require.True(t, model.detailLoading)
	require.NotNil(t, cmd)
}

func TestViewTransitionDetailToLogs(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	// Press 'l' to open logs
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	model := updated.(AppModel)

	require.Equal(t, viewLogs, model.viewMode, "should switch to log view")
	require.Equal(t, "web", model.logService, "should default to web logs")
	require.True(t, model.logLoading, "should be loading logs")
	require.NotNil(t, cmd, "should return a command to load logs")
}

func TestBackFromDetailToDashboard(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	// Press Esc to go back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "esc should return to dashboard")
	require.Nil(t, model.detail, "detail should be cleared")
}

func TestBackFromDetailWithBackspace(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	// Press Backspace to go back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "backspace should return to dashboard")
}

func TestBackFromLogToDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail
	m.logContent = "some log content"

	// Press Esc to go back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "esc from logs should return to detail")
	require.Empty(t, model.logContent, "log content should be cleared")
	require.Equal(t, 0, model.logScroll, "log scroll should be reset")
}

func TestProjectDetailLoadedMsg(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.detailLoading = true

	detail := sampleDetail()
	updated, _ := m.Update(projectDetailLoadedMsg{detail: detail})
	model := updated.(AppModel)

	require.False(t, model.detailLoading, "loading should be false after detail loaded")
	require.NotNil(t, model.detail)
	require.Equal(t, "mysite", model.detail.Name)
	require.Equal(t, "8.2", model.detail.PHPVersion)
	require.Equal(t, "nginx-fpm", model.detail.WebserverType)
	require.Len(t, model.detail.Services, 2)
	require.Len(t, model.detail.Addons, 2)
}

func TestProjectDetailLoadedError(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.detailLoading = true

	updated, _ := m.Update(projectDetailLoadedMsg{err: errTest})
	model := updated.(AppModel)

	require.False(t, model.detailLoading)
	require.Equal(t, viewDashboard, model.viewMode, "should return to dashboard on error")
	require.Contains(t, model.statusMsg, "Error")
}

func TestLogsLoadedMsg(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.logLoading = true

	updated, _ := m.Update(logsLoadedMsg{logs: "line1\nline2\nline3", service: "web"})
	model := updated.(AppModel)

	require.False(t, model.logLoading)
	require.Equal(t, "line1\nline2\nline3", model.logContent)
	require.Equal(t, "web", model.logService)
	require.Equal(t, 0, model.logScroll)
}

func TestLogsLoadedError(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.logLoading = true

	updated, _ := m.Update(logsLoadedMsg{err: errTest, service: "web"})
	model := updated.(AppModel)

	require.False(t, model.logLoading)
	require.Contains(t, model.statusMsg, "Error")
}

func TestLogServiceToggle(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail
	m.logService = "web"
	m.logContent = "web logs"

	// Switch to db logs
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(AppModel)

	require.Equal(t, "db", model.logService)
	require.True(t, model.logLoading)
	require.Empty(t, model.logContent, "content should clear when switching")
	require.NotNil(t, cmd)

	// Switch back to web logs
	model.logLoading = false
	model.logContent = "db logs"
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	model = updated.(AppModel)

	require.Equal(t, "web", model.logService)
	require.True(t, model.logLoading)
	require.NotNil(t, cmd)
}

func TestLogServiceToggleNoOpWhenSame(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail
	m.logService = "web"
	m.logContent = "web logs"

	// Press 'w' when already on web — should be a no-op
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	model := updated.(AppModel)

	require.Equal(t, "web", model.logService)
	require.False(t, model.logLoading, "should not reload when service unchanged")
	require.Nil(t, cmd)
}

func TestLogScrolling(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.height = 10
	// Create content with many lines
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("log line %d", i))
	}
	m.logContent = strings.Join(lines, "\n")

	// Scroll down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(AppModel)
	require.Equal(t, 1, model.logScroll, "j should scroll down")

	// Scroll up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 0, model.logScroll, "k should scroll up")

	// Scroll up at top — should stay at 0
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 0, model.logScroll, "should not scroll below 0")
}

func TestDetailViewRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.width = 80
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()

	require.Contains(t, view, "DDEV Project: mysite", "should contain project name")
	require.Contains(t, view, "drupal", "should contain project type")
	require.Contains(t, view, "8.2", "should contain PHP version")
	require.Contains(t, view, "nginx-fpm", "should contain webserver type")
	require.Contains(t, view, "20", "should contain Node.js version")
	require.Contains(t, view, "web", "should contain docroot")
	require.Contains(t, view, "mariadb:10.11", "should contain database info")
	require.Contains(t, view, "off", "should contain xdebug status")
	require.Contains(t, view, "mutagen", "should contain performance mode")
	require.Contains(t, view, "https://mysite.ddev.site", "should contain URL")
	require.Contains(t, view, "8026", "should contain mailpit URL")
	require.Contains(t, view, "32773", "should contain DB port")
	require.Contains(t, view, "ddev-redis", "should contain add-on")
	require.Contains(t, view, "logs", "should contain logs key hint")
	require.Contains(t, view, "back", "should contain back key hint")
}

func TestDetailViewLoadingRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.detailLoading = true
	m.width = 60

	view := m.View()

	require.Contains(t, view, "Loading project detail", "should show loading message")
}

func TestLogViewRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.width = 80
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail
	m.logService = "web"
	m.logContent = "test log line 1\ntest log line 2"

	view := m.View()

	require.Contains(t, view, "DDEV Logs: mysite (web)", "should contain log title")
	require.Contains(t, view, "test log line 1", "should contain log content")
	require.Contains(t, view, "test log line 2", "should contain log content")
	require.Contains(t, view, "scroll", "should contain scroll hint")
	require.Contains(t, view, "back", "should contain back hint")
}

func TestLogViewLoadingRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.logLoading = true
	m.width = 60
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()

	require.Contains(t, view, "Loading logs", "should show loading message")
}

func TestLogViewEmptyRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.width = 60
	m.logContent = ""
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()

	require.Contains(t, view, "No log output", "should show empty message")
}

func TestDetailActionStart(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Starting mysite")
	require.NotNil(t, cmd)
}

func TestDetailActionStop(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Stopping mysite")
	require.NotNil(t, cmd)
}

func TestDetailActionRestart(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Restarting mysite")
	require.NotNil(t, cmd)
}

func TestDetailActionSSH(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	require.NotNil(t, cmd, "e should return a command to ssh")
}

func TestDetailSSHHintVisible(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.width = 80
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()
	require.Contains(t, view, "ssh", "detail view should show ssh key hint")
}

func TestDetailQuit(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	require.True(t, ok, "q in detail should quit")
}

func TestLogQuit(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	require.True(t, ok, "q in log view should quit")
}

func TestOperationDetailFinishedMsg(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	// Successful operation should reload both detail and project list
	updated, cmd := m.Update(operationDetailFinishedMsg{err: nil})
	model := updated.(AppModel)

	require.Equal(t, "Operation completed", model.statusMsg)
	require.True(t, model.detailLoading, "should reload detail after operation")
	require.True(t, model.loading, "should reload project list after operation")
	require.NotNil(t, cmd)

	// Failed operation
	updated, _ = m.Update(operationDetailFinishedMsg{err: errTest})
	model = updated.(AppModel)
	require.Contains(t, model.statusMsg, "Error")
}

func TestTickRefreshesByViewMode(t *testing.T) {
	// On dashboard, tick should reload projects
	m := NewAppModel()
	m.viewMode = viewDashboard
	_, cmd := m.Update(tickMsg{})
	require.NotNil(t, cmd, "tick on dashboard should return commands")

	// On detail view, tick should refresh detail
	detail := sampleDetail()
	m.detail = &detail
	m.viewMode = viewDetail
	_, cmd = m.Update(tickMsg{})
	require.NotNil(t, cmd, "tick on detail should return commands")

	// On log view, tick should only re-tick
	m.viewMode = viewLogs
	_, cmd = m.Update(tickMsg{})
	require.NotNil(t, cmd, "tick on logs should return tick cmd")
}

func TestDetailNoEntryWithNoProjects(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	// No projects

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "should stay on dashboard with no projects")
	require.Nil(t, cmd)
}

func TestNewAppModelViewMode(t *testing.T) {
	m := NewAppModel()
	require.Equal(t, viewDashboard, m.viewMode, "should start on dashboard")
	require.Nil(t, m.detail, "detail should start nil")
	require.Empty(t, m.logContent, "log content should start empty")
}

func TestDetailRefresh(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	model := updated.(AppModel)

	require.True(t, model.detailLoading, "should set loading on refresh")
	require.Contains(t, model.statusMsg, "Refreshing")
	require.NotNil(t, cmd)
}

func TestLogRefresh(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail
	m.logService = "web"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	model := updated.(AppModel)

	require.True(t, model.logLoading, "should set loading on refresh")
	require.NotNil(t, cmd)
}

// --- Visual polish tests ---

func TestSortServices(t *testing.T) {
	services := []ServiceInfo{
		{Name: "redis", Status: ddevapp.SiteRunning},
		{Name: "db", Status: ddevapp.SiteRunning},
		{Name: "elasticsearch", Status: ddevapp.SiteRunning},
		{Name: "web", Status: ddevapp.SiteRunning},
	}

	sorted := sortServices(services)

	require.Len(t, sorted, 4)
	require.Equal(t, "web", sorted[0].Name, "web should be first")
	require.Equal(t, "db", sorted[1].Name, "db should be second")
	require.Equal(t, "elasticsearch", sorted[2].Name, "rest should be alphabetical")
	require.Equal(t, "redis", sorted[3].Name)

	// Original should not be modified
	require.Equal(t, "redis", services[0].Name, "original slice should not be modified")
}

func TestSortServicesMinimal(t *testing.T) {
	// Only web
	sorted := sortServices([]ServiceInfo{{Name: "web", Status: "running"}})
	require.Len(t, sorted, 1)
	require.Equal(t, "web", sorted[0].Name)

	// Empty
	sorted = sortServices(nil)
	require.Empty(t, sorted)
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "hello", truncate("hello", 10), "short string should not be truncated")
	require.Equal(t, "hello", truncate("hello", 5), "exact length should not be truncated")
	require.Equal(t, "he...", truncate("hello world", 5), "long string should be truncated with ellipsis")
	require.Equal(t, "hel...", truncate("hello world", 6))
	require.Equal(t, "hello world", truncate("hello world", 0), "maxLen 0 should return original")
	require.Equal(t, "he", truncate("hello", 2), "very short maxLen returns prefix")
}

func TestNarrowTerminalDashboard(t *testing.T) {
	m := NewAppModel()
	m.width = 40
	m.height = 24
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "very-long-project-name", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://very-long-project-name.ddev.site"},
	}

	view := m.View()

	// Should still render without URL (narrow hides URLs)
	require.Contains(t, view, "drupal")
	require.NotContains(t, view, "https://", "URL should be hidden in narrow terminal")
}

func TestWideTerminalDashboard(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 24
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site"},
	}

	view := m.View()

	require.Contains(t, view, "https://mysite.ddev.site", "URL should be visible in wide terminal")
}

func TestSpinnerInLoadingView(t *testing.T) {
	m := NewAppModel()
	m.loading = true
	m.width = 60

	view := m.View()
	require.Contains(t, view, "Loading projects", "should show loading text")
}

func TestIsLoading(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.detailLoading = false
	m.logLoading = false
	require.False(t, m.isLoading())

	m.loading = true
	require.True(t, m.isLoading())

	m.loading = false
	m.detailLoading = true
	require.True(t, m.isLoading())

	m.detailLoading = false
	m.logLoading = true
	require.True(t, m.isLoading())
}

// --- Multi-project operations tests ---

func TestStartAllConfirmation(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "a", Status: ddevapp.SiteStopped},
		{Name: "b", Status: ddevapp.SiteStopped},
	}

	// Press S to start all — should enter confirmation
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(AppModel)

	require.True(t, model.confirming, "should enter confirmation mode")
	require.Equal(t, "start-all", model.confirmAction)
	require.Contains(t, model.statusMsg, "Start all 2 projects")
	require.Nil(t, cmd, "should not execute yet")
}

func TestStopAllConfirmation(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "a", Status: ddevapp.SiteRunning},
		{Name: "b", Status: ddevapp.SiteRunning},
	}

	// Press X to stop all — should enter confirmation
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model := updated.(AppModel)

	require.True(t, model.confirming, "should enter confirmation mode")
	require.Equal(t, "stop-all", model.confirmAction)
	require.Contains(t, model.statusMsg, "Stop all 2 projects")
	require.Nil(t, cmd)
}

func TestConfirmStartAll(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{{Name: "a"}}
	m.confirming = true
	m.confirmAction = "start-all"

	// Press y to confirm
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(AppModel)

	require.False(t, model.confirming, "should exit confirmation")
	require.Empty(t, model.confirmAction)
	require.Contains(t, model.statusMsg, "Starting all")
	require.NotNil(t, cmd, "should return exec command")
}

func TestConfirmStopAll(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{{Name: "a"}}
	m.confirming = true
	m.confirmAction = "stop-all"

	// Press y to confirm
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(AppModel)

	require.False(t, model.confirming)
	require.Contains(t, model.statusMsg, "Stopping all")
	require.NotNil(t, cmd)
}

func TestCancelConfirmation(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{{Name: "a"}}
	m.confirming = true
	m.confirmAction = "start-all"
	m.statusMsg = "Start all?"

	// Press any non-y key to cancel
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(AppModel)

	require.False(t, model.confirming, "should cancel confirmation")
	require.Empty(t, model.confirmAction)
	require.Empty(t, model.statusMsg, "status should be cleared")
	require.Nil(t, cmd)
}

func TestStartAllNoProjectsNoop(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	// No projects

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(AppModel)

	require.False(t, model.confirming, "should not confirm with no projects")
}

func TestStartAllHintVisible(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 120
	m.projects = []ProjectInfo{{Name: "a"}}

	view := m.View()

	require.Contains(t, view, "start all", "should show start all hint")
	require.Contains(t, view, "stop all", "should show stop all hint")
}

// --- Error resilience tests ---

func TestProjectsLoadedErrorShowsMessage(t *testing.T) {
	m := NewAppModel()
	m.loading = true

	updated, _ := m.Update(projectsLoadedMsg{err: errTest})
	model := updated.(AppModel)

	require.False(t, model.loading)
	require.Error(t, model.err)
	require.Contains(t, model.statusMsg, "Error loading projects")
}

func TestProjectsLoadedErrorClearsOnSuccess(t *testing.T) {
	m := NewAppModel()
	m.err = errTest
	m.statusMsg = "Error loading projects: test error"

	updated, _ := m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{{Name: "a"}},
	})
	model := updated.(AppModel)

	require.NoError(t, model.err, "error should be cleared on success")
}

func TestErrorViewShowsRetryHint(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.err = errTest
	m.width = 60

	view := m.View()

	require.Contains(t, view, "Error", "should show error")
	require.Contains(t, view, "Press R to retry", "should show retry hint")
}
