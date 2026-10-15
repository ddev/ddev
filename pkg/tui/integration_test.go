package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/testsetup"
	"github.com/stretchr/testify/require"
)

// integration holds two real projects, one running and one stopped. They are
// created on first use so that the unit tests do not need Docker.
var integration struct {
	once  sync.Once
	err   error
	bin   string
	names [2]string
	dirs  [2]string
}

const (
	runningIdx = 0
	stoppedIdx = 1
)

// useIntegrationProjects creates the test projects if needed and points the
// TUI at the ddev binary under test.
func useIntegrationProjects(t *testing.T) {
	t.Helper()
	testcommon.SkipUnlessDefaultEnvironment(t)
	integration.once.Do(setupIntegration)
	require.NoError(t, integration.err)
	orig := ddevBinary
	t.Cleanup(func() { ddevBinary = orig })
	ddevBinary = func() (string, error) { return integration.bin, nil }
}

func setupIntegration() {
	_ = os.Setenv("DDEV_NONINTERACTIVE", "true")
	testcommon.ClearDockerEnv()
	integration.bin, integration.err = testsetup.ResolveDdevBinary()
	if integration.err != nil {
		return
	}
	integration.names = [2]string{"tui-test-running", "tui-test-stopped"}
	for i, name := range integration.names {
		// A run killed before teardown leaves its projects registered, which
		// would make ddev config refuse the new directory.
		_ = runDdev("", "delete", "-Oy", name)
		integration.dirs[i] = testcommon.CreateTmpDir(name)
		if integration.err = runDdev(integration.dirs[i], "config", "--project-type=php", "--project-name="+name); integration.err != nil {
			return
		}
	}
	integration.err = runDdev(integration.dirs[runningIdx], "start", "-y")
}

func teardownIntegration() {
	for i, dir := range integration.dirs {
		if dir == "" {
			continue
		}
		_ = runDdev(dir, "delete", "-Oy", integration.names[i])
		testcommon.CleanupDir(dir)
	}
}

func runDdev(dir string, args ...string) error {
	c := exec.Command(integration.bin, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "DDEV_NO_TUI=true")
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("ddev %s in %s: %w\n%s", strings.Join(args, " "), dir, err, out)
	}
	return nil
}

// integrationProject returns the loaded project i.
func integrationProject(t *testing.T, msg projectsLoadedMsg, i int) ProjectInfo {
	t.Helper()
	for _, p := range msg.projects {
		if p.Name == integration.names[i] {
			return p
		}
	}
	require.Failf(t, "project missing", "%s is not in the project list", integration.names[i])
	return ProjectInfo{}
}

func TestIntegrationLoadProjects(t *testing.T) {
	useIntegrationProjects(t)
	msg, ok := loadProjects().(projectsLoadedMsg)
	require.True(t, ok)
	require.NoError(t, msg.err)
	require.NotEmpty(t, msg.router.status)
	require.NotEmpty(t, msg.sshAgent.status)

	running := integrationProject(t, msg, runningIdx)
	require.Equal(t, ddevapp.SiteRunning, running.Status)
	require.Equal(t, "php", running.Type)
	require.Equal(t, integration.dirs[runningIdx], running.AppRoot)
	require.Contains(t, running.URL, integration.names[runningIdx])

	stopped := integrationProject(t, msg, stoppedIdx)
	require.Equal(t, ddevapp.SiteStopped, stopped.Status)
	require.Empty(t, stopped.URL, "a stopped project has no URL to launch")
}

func TestIntegrationReadProjects(t *testing.T) {
	useIntegrationProjects(t)
	msg, ok := readProjects().(projectsListedMsg)
	require.True(t, ok)
	loaded := projectsLoadedMsg{projects: msg.projects}
	for _, i := range []int{runningIdx, stoppedIdx} {
		p := integrationProject(t, loaded, i)
		require.Equal(t, statusLoading, p.Status, "the status is left to loadProjects")
		require.Equal(t, "php", p.Type)
		require.Equal(t, integration.dirs[i], p.AppRoot)
		require.NotNil(t, p.Detail)
		require.True(t, p.Detail.FromConfig)
		require.NotEmpty(t, p.Detail.PHPVersion)
		require.NotEmpty(t, p.Detail.DatabaseType)
	}
}

// TestIntegrationMissingProject checks that the warning about a project
// whose directory is gone goes to the status line, not over the screen.
func TestIntegrationMissingProject(t *testing.T) {
	useIntegrationProjects(t)
	name := "tui-test-missing"
	_ = runDdev("", "delete", "-Oy", name)
	dir := testcommon.CreateTmpDir(name)
	require.NoError(t, runDdev(dir, "config", "--project-type=php", "--project-name="+name))
	testcommon.CleanupDir(dir)
	t.Cleanup(func() { _ = runDdev("", "delete", "-Oy", name) })

	restore := captureLogs()
	defer restore()
	msg, ok := loadProjects().(projectsLoadedMsg)
	require.True(t, ok)
	require.NoError(t, msg.err)
	require.Contains(t, msg.warning, "no longer exists")
	for _, p := range msg.projects {
		require.NotEqual(t, name, p.Name, "the missing project is removed from the list")
	}
}

func TestIntegrationLoadDetail(t *testing.T) {
	useIntegrationProjects(t)

	running, err := loadDetail(integration.dirs[runningIdx])
	require.NoError(t, err)
	require.Equal(t, integration.names[runningIdx], running.Name)
	require.Equal(t, ddevapp.SiteRunning, running.Status)
	require.NotEmpty(t, running.PHPVersion)
	require.NotEmpty(t, running.WebserverType)
	require.NotEmpty(t, running.DatabaseType)
	require.NotEmpty(t, running.URLs)
	require.NotEmpty(t, running.MailpitURL)
	require.Regexp(t, `:\d+$`, running.DBHostPort)
	require.Equal(t, "db:3306", running.DBInternal)
	require.Equal(t, "ddev-"+integration.names[runningIdx]+"-db:3306", running.DBShared)
	require.NotEmpty(t, running.Services[0].Ports)
	require.GreaterOrEqual(t, len(running.Services), 2)
	require.Equal(t, "web", running.Services[0].Name)
	require.Equal(t, "db", running.Services[1].Name)

	stopped, err := loadDetail(integration.dirs[stoppedIdx])
	require.NoError(t, err)
	require.Equal(t, ddevapp.SiteStopped, stopped.Status)
	require.Empty(t, stopped.DBHostPort, "a stopped project publishes no database port")

	_, err = loadDetail(integration.dirs[runningIdx] + "-missing")
	require.ErrorContains(t, err, "no longer exists")
}

func TestIntegrationXdebug(t *testing.T) {
	useIntegrationProjects(t)
	for _, step := range []struct {
		arg     string
		enabled bool
	}{{"status", false}, {"on", true}, {"status", true}, {"off", false}, {"status", false}} {
		msg, ok := xdebugCmd(integration.dirs[runningIdx], step.arg)().(xdebugStatusMsg)
		require.True(t, ok)
		require.NoError(t, msg.err, step.arg)
		require.Equal(t, step.enabled, msg.enabled, "after xdebug %s", step.arg)
	}

	msg, ok := xdebugCmd(integration.dirs[stoppedIdx], "status")().(xdebugStatusMsg)
	require.True(t, ok)
	require.Error(t, msg.err)
	app, err := ddevapp.NewApp(integration.dirs[stoppedIdx], true)
	require.NoError(t, err)
	status, _ := app.SiteStatus()
	require.NotEqual(t, ddevapp.SiteRunning, status, "unlike ddev xdebug, asking does not start a stopped project")
}

func TestIntegrationViews(t *testing.T) {
	useIntegrationProjects(t)
	m := newModel(t, 120, 30)
	// Other projects on the machine can push the test projects off the
	// first page.
	m = filterTo(t, send(t, m, loadProjects()), "tui-test-")
	view := screen(m)
	for _, name := range integration.names {
		require.Contains(t, view, name)
	}
	require.Contains(t, view, ddevapp.SiteRunning)
	require.Contains(t, view, ddevapp.SiteStopped)
	requireFits(t, m, "dashboard")

	m, _ = m.openDetail(ProjectInfo{Name: integration.names[runningIdx], AppRoot: integration.dirs[runningIdx]})
	m = send(t, m, loadDetailCmd(integration.dirs[runningIdx])())
	view = screen(m)
	require.Contains(t, view, m.detail.project.PHPVersion)
	require.Contains(t, view, m.detail.project.DBHostPort)
	require.Contains(t, view, "web")
	requireFits(t, m, "detail")
}

// TestIntegrationProgram runs the TUI as a program and filters the list down
// to the stopped project.
func TestIntegrationProgram(t *testing.T) {
	useIntegrationProjects(t)
	loaded := make(chan struct{})
	var once sync.Once
	p := tea.NewProgram(NewAppModel(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
		tea.WithWindowSize(120, 30),
		tea.WithoutSignals(),
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(projectsLoadedMsg); ok {
				once.Do(func() { close(loaded) })
			}
			return msg
		}),
	)
	go func() {
		select {
		case <-loaded:
		case <-time.After(2 * time.Minute):
		}
		p.Send(press("/"))
		for _, r := range "tui-test-stop" {
			p.Send(press(string(r)))
		}
		p.Send(press("enter"))
		p.Send(press("q"))
	}()
	final, err := p.Run()
	require.NoError(t, err)
	m := final.(AppModel)
	require.Equal(t, "tui-test-stop", m.dashboard.filter.Value())
	require.NotNil(t, m.dashboard.selected())
	require.Equal(t, integration.names[stoppedIdx], m.dashboard.selected().Name)
}
