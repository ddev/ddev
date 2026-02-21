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
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m = updated.(AppModel)

	// Then load projects
	m.loading = false
	updated, _ = m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{
			{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site"},
			{Name: "other", Status: ddevapp.SiteStopped, Type: "wordpress"},
		},
	})
	m = updated.(AppModel)

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
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m = updated.(AppModel)

	// Then load empty projects
	m.loading = false
	updated, _ = m.Update(projectsLoadedMsg{projects: []ProjectInfo{}})
	m = updated.(AppModel)

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

func TestDetailActionLogs(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	// Press 'L' to open streaming log view
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	model := updated.(AppModel)

	require.Equal(t, viewLogs, model.viewMode, "should switch to log view")
	require.NotNil(t, cmd, "L should return a command to start log stream")
	require.Empty(t, model.logLines, "log lines should start empty")
}

func TestLogStreamMessages(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail

	ch := make(chan string, 10)
	ch <- "line1"

	// Simulate stream started
	updated, cmd := m.Update(logStreamStartedMsg{lines: ch, process: nil})
	model := updated.(AppModel)
	require.NotNil(t, model.logSub, "should store log channel")
	require.NotNil(t, cmd, "should return cmd to wait for lines")

	// Simulate receiving a log line
	updated, cmd = model.Update(logLineMsg{line: "hello from logs"})
	model = updated.(AppModel)
	require.Len(t, model.logLines, 1)
	require.Equal(t, "hello from logs", model.logLines[0])
	require.NotNil(t, cmd, "should return cmd to wait for next line")

	// Simulate stream ended
	updated, _ = model.Update(logStreamEndedMsg{})
	model = updated.(AppModel)
	require.Nil(t, model.logProcess, "process should be cleared")
	require.Nil(t, model.logSub, "channel should be cleared")
}

func TestBackFromLogToDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	detail := sampleDetail()
	m.detail = &detail
	m.logLines = []string{"some log"}

	// Press Esc to go back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "esc from logs should return to detail")
	require.Nil(t, model.logLines, "log lines should be cleared")
	require.Nil(t, model.logSub, "log channel should be cleared")
}

func TestLogViewRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.width = 80
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail
	m.logLines = []string{"log line 1", "log line 2", "log line 3"}

	view := m.View()

	require.Contains(t, view, "DDEV Logs: mysite", "should contain log title")
	require.Contains(t, view, "log line 1", "should contain log content")
	require.Contains(t, view, "log line 3", "should contain log content")
	require.Contains(t, view, "back", "should contain back hint")
}

func TestLogViewWaitingRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.width = 60
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()

	require.Contains(t, view, "Waiting for log output", "should show waiting message")
}

func TestLogViewAutoScrollsToBottom(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewLogs
	m.width = 80
	m.height = 10 // small height: viewHeight = 10 - 4 = 6

	detail := sampleDetail()
	m.detail = &detail

	// Add more lines than fit
	for i := 0; i < 20; i++ {
		m.logLines = append(m.logLines, fmt.Sprintf("line %d", i))
	}

	view := m.View()

	// Should show the last lines, not the first
	require.Contains(t, view, "line 19", "should show last line")
	require.NotContains(t, view, "line 0", "should not show first line")
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

func TestDetailViewRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(AppModel)

	// Then load detail
	detail := sampleDetail()
	updated, _ = m.Update(projectDetailLoadedMsg{detail: detail})
	m = updated.(AppModel)

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
	require.Contains(t, view, "launch", "should contain launch key hint")
	require.Contains(t, view, "mailpit", "should contain mailpit key hint")
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

func TestDetailActionStart(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Starting mysite")
	require.Equal(t, viewDetail, model.operationReturnView)
	require.NotNil(t, cmd)
}

func TestDetailActionStop(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Stopping mysite")
	require.Equal(t, viewDetail, model.operationReturnView)
	require.NotNil(t, cmd)
}

func TestDetailActionXHGui(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd, "x should return a command to launch xhgui")
}

func TestDashboardActionXHGui(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site", AppRoot: "/tmp/mysite"},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd, "x should return a command to launch xhgui from dashboard")
}

func TestDetailActionRestart(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Restarting mysite")
	require.Equal(t, viewDetail, model.operationReturnView)
	require.NotNil(t, cmd)
}

func TestDetailActionLaunch(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	require.NotNil(t, cmd, "l should return a command to launch")
}

func TestDetailActionMailpit(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	require.NotNil(t, cmd, "m should return a command to launch mailpit")
}

func TestDetailActionMailpitNoURL(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	detail.MailpitURL = ""
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	require.Nil(t, cmd, "m should be no-op when no mailpit URL")
}

func TestDashboardActionLaunch(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site", AppRoot: "/tmp/mysite"},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	require.NotNil(t, cmd, "l should return a command to launch from dashboard")
}

func TestDashboardActionMailpit(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site", AppRoot: "/tmp/mysite"},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	require.NotNil(t, cmd, "m should return a command to launch mailpit from dashboard")
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
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m = updated.(AppModel)

	// Then load projects
	m.loading = false
	updated, _ = m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{
			{Name: "very-long-project-name", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://very-long-project-name.ddev.site"},
		},
	})
	m = updated.(AppModel)

	view := m.View()

	// Should still render without URL (narrow hides URLs)
	require.Contains(t, view, "drupal")
	require.NotContains(t, view, "https://", "URL should be hidden in narrow terminal")
}

func TestWideTerminalDashboard(t *testing.T) {
	m := NewAppModel()
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = updated.(AppModel)

	// Then load projects
	m.loading = false
	updated, _ = m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{
			{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", URL: "https://mysite.ddev.site"},
		},
	})
	m = updated.(AppModel)

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
	require.False(t, m.isLoading())

	m.loading = true
	require.True(t, m.isLoading())

	m.loading = false
	m.detailLoading = true
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

	// Press 'a' to start all — should enter confirmation
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
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

	// Press 'A' to stop all — should enter confirmation
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
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
	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Starting all")
	require.NotNil(t, cmd, "should return stream command")
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
	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Stopping all")
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

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
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

func TestAllStoppedHint(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 120
	m.projects = []ProjectInfo{
		{Name: "a", Status: ddevapp.SiteStopped},
		{Name: "b", Status: ddevapp.SiteStopped},
	}

	view := m.View()
	require.Contains(t, view, "All projects stopped", "should show all-stopped hint")
	require.Contains(t, view, "to start selected", "should show start hint")
}

func TestAllStoppedHintHiddenWhenRunning(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 120
	m.projects = []ProjectInfo{
		{Name: "a", Status: ddevapp.SiteRunning},
		{Name: "b", Status: ddevapp.SiteStopped},
	}

	view := m.View()
	require.NotContains(t, view, "All projects are stopped", "should not show all-stopped hint when some are running")
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

func TestRefreshStatusClearsOnProjectsLoaded(t *testing.T) {
	m := NewAppModel()
	m.loading = true
	m.statusMsg = "Refreshing..."

	updated, _ := m.Update(projectsLoadedMsg{
		projects: []ProjectInfo{{Name: "a"}},
	})
	model := updated.(AppModel)

	require.Empty(t, model.statusMsg, "Refreshing status should clear on success")
}

func TestRefreshStatusClearsOnDetailLoaded(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.detailLoading = true
	m.statusMsg = "Refreshing..."
	detail := sampleDetail()

	updated, _ := m.Update(projectDetailLoadedMsg{detail: detail})
	model := updated.(AppModel)

	require.Empty(t, model.statusMsg, "Refreshing status should clear on detail success")
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

// --- Router status tests ---

func TestRouterStatusMsg(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(routerStatusMsg{status: "healthy"})
	model := updated.(AppModel)

	require.Equal(t, "healthy", model.routerStatus, "router status should be set")
}

func TestRouterStatusInDashboardView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 80
	m.routerStatus = "healthy"
	m.projects = []ProjectInfo{{Name: "a", Status: ddevapp.SiteRunning, Type: "php"}}

	view := m.View()
	require.Contains(t, view, "Router:", "dashboard should show router status label")
	require.Contains(t, view, "healthy", "dashboard should show router status value")
}

func TestRouterStatusEmptyNotShown(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 80
	m.routerStatus = ""
	m.projects = []ProjectInfo{{Name: "a", Status: ddevapp.SiteRunning, Type: "php"}}

	view := m.View()
	require.NotContains(t, view, "Router:", "should not show router label when empty")
}

// --- Xdebug toggle tests ---

func TestXdebugToggleFromDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	detail.Status = ddevapp.SiteRunning
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Toggling xdebug")
	require.NotNil(t, cmd, "X should return a command to toggle xdebug")
}

func TestXdebugToggleIgnoredWhenStopped(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	detail.Status = ddevapp.SiteStopped
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	require.Nil(t, cmd, "X should be no-op when project is stopped")
}

func TestXdebugToggledMsgEnabled(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(xdebugToggledMsg{err: nil, enabled: true})
	model := updated.(AppModel)

	require.Equal(t, "Xdebug enabled", model.statusMsg)
	require.True(t, model.detailLoading, "should reload detail after xdebug toggle")
	require.NotNil(t, cmd)
}

func TestXdebugToggledMsgDisabled(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(xdebugToggledMsg{err: nil, enabled: false})
	model := updated.(AppModel)

	require.Equal(t, "Xdebug disabled", model.statusMsg)
	require.True(t, model.detailLoading, "should reload detail after xdebug toggle")
	require.NotNil(t, cmd)
}

func TestXdebugToggledMsgError(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, _ := m.Update(xdebugToggledMsg{err: errTest})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Xdebug toggle failed")
}

func TestXdebugHintInDetailView(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.width = 120
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()
	require.Contains(t, view, "xdebug", "detail view should show xdebug key hint")
}

// --- Poweroff tests ---

func TestPoweroffConfirmation(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{{Name: "a"}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	model := updated.(AppModel)

	require.True(t, model.confirming, "should enter confirmation mode")
	require.Equal(t, "poweroff", model.confirmAction)
	require.Contains(t, model.statusMsg, "Poweroff")
	require.Nil(t, cmd, "should not execute yet")
}

func TestConfirmPoweroff(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.confirming = true
	m.confirmAction = "poweroff"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(AppModel)

	require.False(t, model.confirming, "should exit confirmation")
	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Powering off")
	require.NotNil(t, cmd, "should return stream command")
}

func TestPoweroffHintInDashboard(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 120
	m.projects = []ProjectInfo{{Name: "a"}}

	view := m.View()
	require.Contains(t, view, "poweroff", "dashboard should show poweroff hint")
}

func TestPoweroffInHelpView(t *testing.T) {
	m := NewAppModel()
	m.showHelp = true

	view := m.View()
	require.Contains(t, view, "Poweroff", "help should mention Poweroff")
}

// --- Copy URL tests ---

func TestCopyURLFromDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Copying URL")
	require.NotNil(t, cmd, "c should return a clipboard command")
}

func TestCopyURLNoURLsNoop(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	detail := sampleDetail()
	detail.URLs = nil
	m.detail = &detail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.Nil(t, cmd, "c should be no-op when no URLs")
}

func TestClipboardMsgSuccess(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(clipboardMsg{err: nil})
	model := updated.(AppModel)

	require.Equal(t, "URL copied to clipboard", model.statusMsg)
}

func TestClipboardMsgError(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(clipboardMsg{err: errTest})
	model := updated.(AppModel)

	require.Contains(t, model.statusMsg, "Copy failed")
}

func TestCopyURLHintInDetailView(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewDetail
	m.width = 120
	m.height = 30
	detail := sampleDetail()
	m.detail = &detail

	view := m.View()
	require.Contains(t, view, "copy url", "detail view should show copy url hint")
}

// --- Help view updated content ---

// --- Config key tests ---

func TestConfigKeyReturnsCommand(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{{Name: "a"}}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	require.NotNil(t, cmd, "C should return a command to run ddev config")
}

func TestConfigKeyFromEmptyState(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	// No projects

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	require.NotNil(t, cmd, "C should return a command even with no projects")
}

func TestEmptyStateShowsConfigHint(t *testing.T) {
	m := NewAppModel()
	// Initialize viewport first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(AppModel)

	// Then load empty projects
	m.loading = false
	updated, _ = m.Update(projectsLoadedMsg{projects: []ProjectInfo{}})
	m = updated.(AppModel)

	view := m.View()
	require.Contains(t, view, "Press 'C' to run ddev config", "empty state should mention C key")
}

func TestConfigHintInDashboard(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.width = 120
	m.projects = []ProjectInfo{{Name: "a"}}

	view := m.View()
	require.Contains(t, view, "config", "dashboard hints should include config")
}

func TestConfigInHelpView(t *testing.T) {
	m := NewAppModel()
	m.showHelp = true

	view := m.View()
	require.Contains(t, view, "Run ddev config", "help should mention ddev config")
}

func TestHelpViewNewEntries(t *testing.T) {
	m := NewAppModel()
	m.showHelp = true

	view := m.View()
	require.Contains(t, view, "Poweroff", "help should mention Poweroff")
	require.Contains(t, view, "Xdebug", "help should mention Xdebug")
	require.Contains(t, view, "clipboard", "help should mention clipboard/copy")
}

// --- Operation streaming view tests ---

func TestDashboardStartEntersOperationView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", AppRoot: "/tmp/mysite"},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Starting mysite")
	require.Equal(t, viewDashboard, model.operationReturnView)
	require.NotNil(t, cmd)
}

func TestDashboardStopEntersOperationView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.projects = []ProjectInfo{
		{Name: "mysite", Status: ddevapp.SiteRunning, Type: "drupal", AppRoot: "/tmp/mysite"},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode)
	require.Contains(t, model.operationName, "Stopping mysite")
	require.Equal(t, viewDashboard, model.operationReturnView)
	require.NotNil(t, cmd)
}

func TestOperationStreamStartedMsg(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"

	ch := make(chan string, 10)
	errCh := make(chan error, 1)

	updated, cmd := m.Update(operationStreamStartedMsg{lines: ch, errCh: errCh, process: nil})
	model := updated.(AppModel)

	require.NotNil(t, model.logSub, "should store line channel")
	require.NotNil(t, model.operationErrCh, "should store error channel")
	require.NotNil(t, cmd, "should return cmd to wait for lines")
}

func TestOperationStreamEndedMsgSuccess(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.operationReturnView = viewDashboard

	updated, cmd := m.Update(operationStreamEndedMsg{err: nil})
	model := updated.(AppModel)

	require.True(t, model.operationDone)
	require.NoError(t, model.operationErr)
	require.Nil(t, model.logProcess)
	require.Nil(t, model.logSub)
	require.Nil(t, model.operationErrCh)
	require.NotNil(t, cmd, "should return reload commands")
}

func TestOperationStreamEndedMsgError(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"

	updated, _ := m.Update(operationStreamEndedMsg{err: errTest})
	model := updated.(AppModel)

	require.True(t, model.operationDone)
	require.Error(t, model.operationErr)
}

func TestOperationStreamEndedReturnsToDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationReturnView = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(operationStreamEndedMsg{err: nil})
	model := updated.(AppModel)

	require.True(t, model.operationDone)
	require.NotNil(t, cmd, "should return reload commands including detail")
}

func TestLogLineMsgInOperationView(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"

	ch := make(chan string, 10)
	errCh := make(chan error, 1)
	m.logSub = ch
	m.operationErrCh = errCh

	updated, cmd := m.Update(logLineMsg{line: "Building container..."})
	model := updated.(AppModel)

	require.Len(t, model.logLines, 1)
	require.Equal(t, "Building container...", model.logLines[0])
	require.NotNil(t, cmd, "should return cmd to wait for next line via operation channel")
}

func TestOperationViewRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.width = 80
	m.height = 30
	m.logLines = []string{"Building...", "Starting web container..."}

	view := m.View()

	require.Contains(t, view, "Starting mysite", "should contain operation title")
	require.Contains(t, view, "Building...", "should contain output")
	require.Contains(t, view, "Starting web container...", "should contain output")
	require.Contains(t, view, "back", "should contain back hint")
	require.Contains(t, view, "quit", "should contain quit hint")
}

func TestOperationViewDoneRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.operationDone = true
	m.width = 80
	m.height = 30
	m.logLines = []string{"Done."}

	// Success
	view := m.View()
	require.Contains(t, view, "Completed", "should show completed status")

	// Failure
	m.operationErr = errTest
	view = m.View()
	require.Contains(t, view, "Failed", "should show failed status")
}

func TestOperationViewRunningRendering(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.width = 60

	view := m.View()
	require.Contains(t, view, "Running...", "should show running message when no output")
}

func TestOperationKeyEscBack(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.operationReturnView = viewDashboard
	m.logLines = []string{"some output"}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "esc should return to dashboard")
	require.Nil(t, model.logLines)
	require.Empty(t, model.operationName)
	require.False(t, model.operationDone)
	require.NotNil(t, cmd, "should return reload commands")
}

func TestOperationKeyEscBackToDetail(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"
	m.operationReturnView = viewDetail
	detail := sampleDetail()
	m.detail = &detail

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "esc should return to detail")
	require.True(t, model.detailLoading, "should reload detail")
	require.NotNil(t, cmd)
}

func TestOperationKeyQuit(t *testing.T) {
	m := NewAppModel()
	m.viewMode = viewOperation
	m.operationName = "Starting mysite"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	require.True(t, ok, "q in operation view should quit")
}

func TestEnterOperationView(t *testing.T) {
	m := NewAppModel()
	m.logLines = []string{"old"}
	m.statusMsg = "old status"
	m.operationDone = true
	m.operationErr = errTest

	m = m.enterOperationView("Starting mysite", viewDetail)

	require.Equal(t, viewOperation, m.viewMode)
	require.Equal(t, "Starting mysite", m.operationName)
	require.Equal(t, viewDetail, m.operationReturnView)
	require.Nil(t, m.logLines)
	require.Nil(t, m.logProcess)
	require.Nil(t, m.logSub)
	require.Nil(t, m.operationErrCh)
	require.False(t, m.operationDone)
	require.NoError(t, m.operationErr)
	require.Empty(t, m.statusMsg)
}

func TestIsLoadingWithOperationView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewOperation
	m.operationDone = false
	require.True(t, m.isLoading(), "should be loading during active operation")

	m.operationDone = true
	require.False(t, m.isLoading(), "should not be loading after operation done")
}

func TestLogLineMsgCapInOperationView(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewOperation

	// Add 1001 lines to trigger the cap
	for i := 0; i < 1001; i++ {
		m.logLines = append(m.logLines, fmt.Sprintf("line %d", i))
	}
	require.Equal(t, 1001, len(m.logLines))

	// Simulate receiving one more logLineMsg which triggers the cap logic
	m.logLines = append(m.logLines, "one more line")
	if len(m.logLines) > 1000 {
		m.logLines = m.logLines[len(m.logLines)-500:]
	}
	require.Equal(t, 500, len(m.logLines))
	require.Equal(t, "one more line", m.logLines[len(m.logLines)-1])
}

func TestOperationAutoReturnOnSuccess(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewOperation
	m.operationDone = true
	m.operationErr = nil
	m.operationName = "Starting mysite"
	m.operationReturnView = viewDashboard
	m.logLines = []string{"line1"}

	updated, _ := m.Update(operationAutoReturnMsg{})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "should return to dashboard")
	require.False(t, model.operationDone)
	require.Empty(t, model.operationName)
	require.Nil(t, model.logLines)
	require.Equal(t, "Operation completed", model.statusMsg)
}

func TestOperationAutoReturnToDetail(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewOperation
	m.operationDone = true
	m.operationErr = nil
	m.operationReturnView = viewDetail

	updated, _ := m.Update(operationAutoReturnMsg{})
	model := updated.(AppModel)

	require.Equal(t, viewDetail, model.viewMode, "should return to detail view")
}

func TestOperationAutoReturnSkippedOnError(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewOperation
	m.operationDone = true
	m.operationErr = errTest
	m.operationReturnView = viewDashboard

	updated, _ := m.Update(operationAutoReturnMsg{})
	model := updated.(AppModel)

	require.Equal(t, viewOperation, model.viewMode, "should stay on operation view on error")
	require.True(t, model.operationDone)
}

func TestOperationAutoReturnIgnoredIfAlreadyLeft(t *testing.T) {
	m := NewAppModel()
	m.loading = false
	m.viewMode = viewDashboard // User already pressed esc
	m.operationDone = false

	updated, _ := m.Update(operationAutoReturnMsg{})
	model := updated.(AppModel)

	require.Equal(t, viewDashboard, model.viewMode, "should stay on dashboard")
}
