package tui

import (
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
