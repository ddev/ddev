//go:build integration

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// testApps holds the two test projects created in TestMain.
var testApps [2]*ddevapp.DdevApp

// testDirs holds the temporary directories for cleanup.
var testDirs [2]string

// testNames holds the project names, derived from TestTUIIntegration.
var testNames = [2]string{"TestTUIIntegrationAlpha", "TestTUIIntegrationBeta"}

func TestMain(m *testing.M) {
	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	_ = os.Setenv("DDEV_NO_INSTRUMENTATION", "true")
	_ = os.Setenv("DOCKER_CLI_HINTS", "false")
	_ = os.Setenv("MUTAGEN_DATA_DIRECTORY", globalconfig.GetMutagenDataDirectory())

	globalconfig.EnsureGlobalConfig()
	testcommon.ClearDockerEnv()
	dockerutil.EnsureDdevNetwork()

	for i, name := range testNames {
		// Clean up any leftovers from previous runs
		oldApp := &ddevapp.DdevApp{Name: name}
		_ = oldApp.Stop(true, false)
		_ = globalconfig.RemoveProjectInfo(name)

		testDirs[i] = testcommon.CreateTmpDir(name)

		app, err := ddevapp.NewApp(testDirs[i], true)
		if err != nil {
			teardownAll()
			os.Exit(1)
		}
		app.Name = name
		app.Type = nodeps.AppTypePHP

		err = app.WriteConfig()
		if err != nil {
			teardownAll()
			os.Exit(1)
		}

		// Write a trivial index.php so the docroot is not empty
		_ = os.WriteFile(filepath.Join(testDirs[i], "index.php"), []byte("<?php echo 'ok';"), 0644)

		err = ddevapp.PrepDdevDirectory(app)
		if err != nil {
			teardownAll()
			os.Exit(1)
		}

		err = ddevapp.PopulateExamplesCommandsHomeadditions(app.Name)
		if err != nil {
			teardownAll()
			os.Exit(1)
		}

		err = app.Start()
		if err != nil {
			teardownAll()
			os.Exit(1)
		}
		testApps[i] = app
	}

	testRun := m.Run()

	teardownAll()
	os.Exit(testRun)
}

func teardownAll() {
	for i, app := range testApps {
		if app != nil {
			_ = app.Stop(true, false)
			_ = globalconfig.RemoveProjectInfo(app.Name)
		}
		if testDirs[i] != "" {
			_ = os.RemoveAll(testDirs[i])
		}
	}
}

// TestTUIIntegrationLoadProjects calls loadProjects() and verifies both test
// projects appear with the expected fields.
func TestTUIIntegrationLoadProjects(t *testing.T) {
	msg := loadProjects()
	plMsg, ok := msg.(projectsLoadedMsg)
	require.True(t, ok, "loadProjects should return projectsLoadedMsg")
	require.NoError(t, plMsg.err)
	require.GreaterOrEqual(t, len(plMsg.projects), 2, "should have at least 2 projects")

	foundAlpha := false
	foundBeta := false
	for _, p := range plMsg.projects {
		switch p.Name {
		case testNames[0]:
			foundAlpha = true
			require.Equal(t, ddevapp.SiteRunning, p.Status)
			require.Equal(t, nodeps.AppTypePHP, p.Type)
			require.NotEmpty(t, p.URL, "running project should have a URL")
			require.Equal(t, testDirs[0], p.AppRoot)
		case testNames[1]:
			foundBeta = true
			require.Equal(t, ddevapp.SiteRunning, p.Status)
			require.Equal(t, nodeps.AppTypePHP, p.Type)
			require.NotEmpty(t, p.URL)
			require.Equal(t, testDirs[1], p.AppRoot)
		}
	}
	require.True(t, foundAlpha, "%s should be in the project list", testNames[0])
	require.True(t, foundBeta, "%s should be in the project list", testNames[1])
}

// TestTUIIntegrationDashboardRender feeds a real projectsLoadedMsg into the
// model and verifies the dashboard view contains expected content.
func TestTUIIntegrationDashboardRender(t *testing.T) {
	msg := loadProjects()
	plMsg := msg.(projectsLoadedMsg)
	require.NoError(t, plMsg.err)

	m := NewAppModel()
	m.width = 120
	m.height = 40

	updated, _ := m.Update(plMsg)
	model := updated.(AppModel)
	require.False(t, model.loading)

	view := model.View()
	require.Contains(t, view, "DDEV Projects", "dashboard should contain title")
	require.Contains(t, view, testNames[0], "dashboard should contain first project name")
	require.Contains(t, view, testNames[1], "dashboard should contain second project name")
	require.Contains(t, view, ddevapp.SiteRunning, "dashboard should show running status")
	require.Contains(t, view, nodeps.AppTypePHP, "dashboard should show project type")
	require.Contains(t, view, "start", "dashboard should show key hints")
	require.Contains(t, view, "quit", "dashboard should show quit hint")

	// At least one URL should appear since projects are running and terminal is wide
	require.Contains(t, view, ".ddev.site", "dashboard should show a project URL")
}

// TestTUIIntegrationLoadDetail calls loadDetailCmd for a running project and
// verifies that detail fields are populated.
func TestTUIIntegrationLoadDetail(t *testing.T) {
	cmd := loadDetailCmd(testDirs[0])
	msg := cmd()

	detailMsg, ok := msg.(projectDetailLoadedMsg)
	require.True(t, ok, "loadDetailCmd should return projectDetailLoadedMsg")
	require.NoError(t, detailMsg.err)

	d := detailMsg.detail
	require.Equal(t, testNames[0], d.Name)
	require.Equal(t, ddevapp.SiteRunning, d.Status)
	require.Equal(t, nodeps.AppTypePHP, d.Type)
	require.NotEmpty(t, d.PHPVersion, "PHP version should be populated")
	require.NotEmpty(t, d.WebserverType, "webserver type should be populated")
	require.NotEmpty(t, d.DatabaseType, "database type should be populated")
	require.NotEmpty(t, d.URLs, "running project should have URLs")
	require.Equal(t, testDirs[0], d.AppRoot)

	// Verify core services exist
	serviceNames := make([]string, 0, len(d.Services))
	for _, svc := range d.Services {
		serviceNames = append(serviceNames, svc.Name)
	}
	require.Contains(t, serviceNames, "web", "services should include web")
	require.Contains(t, serviceNames, "db", "services should include db")
}

// TestTUIIntegrationDetailViewRender feeds a real projectDetailLoadedMsg into
// the model and verifies the detail view renders correctly.
func TestTUIIntegrationDetailViewRender(t *testing.T) {
	cmd := loadDetailCmd(testDirs[0])
	detailMsg := cmd().(projectDetailLoadedMsg)
	require.NoError(t, detailMsg.err)

	m := NewAppModel()
	m.viewMode = viewDetail
	m.detailLoading = true
	m.width = 120
	m.height = 40

	updated, _ := m.Update(detailMsg)
	model := updated.(AppModel)
	require.False(t, model.detailLoading)
	require.NotNil(t, model.detail)

	view := model.View()
	require.Contains(t, view, "DDEV Project: "+testNames[0], "detail view should contain project name")
	require.Contains(t, view, nodeps.AppTypePHP, "detail view should contain project type")
	require.Contains(t, view, model.detail.PHPVersion, "detail view should contain PHP version")
	require.Contains(t, view, model.detail.WebserverType, "detail view should contain webserver type")
	require.Contains(t, view, "web", "detail view should show web service")
	require.Contains(t, view, "db", "detail view should show db service")
	require.Contains(t, view, "back", "detail view should show back hint")
	require.Contains(t, view, "logs", "detail view should show logs hint")
	require.Contains(t, view, "ssh", "detail view should show ssh hint")
}

// TestTUIIntegrationDashboardNavigation loads real projects, then navigates
// with j/k and verifies cursor movement and filtering.
func TestTUIIntegrationDashboardNavigation(t *testing.T) {
	msg := loadProjects()
	plMsg := msg.(projectsLoadedMsg)
	require.NoError(t, plMsg.err)

	m := NewAppModel()
	m.width = 120
	m.height = 40
	updated, _ := m.Update(plMsg)
	model := updated.(AppModel)
	require.GreaterOrEqual(t, len(model.projects), 2)

	// Cursor starts at 0
	require.Equal(t, 0, model.cursor)
	firstName := model.filteredProjects()[0].Name

	// Move down
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(AppModel)
	require.Equal(t, 1, model.cursor)
	secondName := model.filteredProjects()[1].Name
	require.NotEqual(t, firstName, secondName, "different cursor positions should select different projects")

	// Move back up
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(AppModel)
	require.Equal(t, 0, model.cursor)

	// Enter filter mode and type a partial name to narrow results
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = updated.(AppModel)
	require.True(t, model.filtering)

	// Type "alpha" (case-insensitive match against testNames[0])
	for _, ch := range "alpha" {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		model = updated.(AppModel)
	}

	filtered := model.filteredProjects()
	require.Len(t, filtered, 1, "filter should match exactly one project")
	require.Equal(t, testNames[0], filtered[0].Name)

	// Press enter to confirm filter
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(AppModel)
	require.False(t, model.filtering)
	require.Equal(t, "alpha", model.filterText)

	// Esc clears the filter
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(AppModel)
	require.Empty(t, model.filterText)
	require.GreaterOrEqual(t, len(model.filteredProjects()), 2, "all projects should be visible again")
}

// TestTUIIntegrationExtractProjectInfo calls extractProjectInfo with a real
// DdevApp and verifies the returned fields.
func TestTUIIntegrationExtractProjectInfo(t *testing.T) {
	app := testApps[0]
	info := extractProjectInfo(app)

	require.Equal(t, testNames[0], info.Name)
	require.Equal(t, ddevapp.SiteRunning, info.Status)
	require.Equal(t, nodeps.AppTypePHP, info.Type)
	require.NotEmpty(t, info.URL, "running project should have a URL")
	require.Contains(t, info.URL, strings.ToLower(testNames[0]), "URL should contain the lowercased project name")
	require.Equal(t, testDirs[0], info.AppRoot)
}

// TestTUIIntegrationDetailLoadMissingDir calls loadDetailCmd with a
// nonexistent directory and verifies the error message.
func TestTUIIntegrationDetailLoadMissingDir(t *testing.T) {
	cmd := loadDetailCmd("/nonexistent/path/that/does/not/exist")
	msg := cmd()

	detailMsg, ok := msg.(projectDetailLoadedMsg)
	require.True(t, ok)
	require.Error(t, detailMsg.err)
	require.Contains(t, detailMsg.err.Error(), "no longer exists")
}
