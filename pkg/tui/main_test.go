package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

// fakeDdevEnv makes the test binary act as ddev; see fakeDdev.
const fakeDdevEnv = "TUI_TEST_FAKE_DDEV"

func TestMain(m *testing.M) {
	if script := os.Getenv(fakeDdevEnv); script != "" {
		os.Exit(fakeDdev(script, os.Args[1:]))
	}
	// HasTermHyperlinks caches its result on first use, so this must be set
	// before any test renders a link.
	_ = os.Setenv("FORCE_HYPERLINK", "1")
	// os.Executable is the test binary, which would run the tests again.
	ddevBinary = func() (string, error) {
		return "", fmt.Errorf("unit tests run no ddev; use useFakeDdev or useIntegrationProjects")
	}
	code := m.Run()
	teardownIntegration()
	os.Exit(code)
}

// fakeDdev prints what a test stored for the given ddev arguments: the
// variable TUI_TEST_OUT_<args> holds stdout, TUI_TEST_ERR_<args> stderr, and
// TUI_TEST_EXIT_<args> the exit code.
func fakeDdev(_ string, args []string) int {
	k := fakeKey(args)
	fmt.Fprint(os.Stdout, os.Getenv("TUI_TEST_OUT_"+k))
	fmt.Fprint(os.Stderr, os.Getenv("TUI_TEST_ERR_"+k))
	if os.Getenv("TUI_TEST_HANG_"+k) != "" {
		time.Sleep(time.Minute)
	}
	if os.Getenv("TUI_TEST_EXIT_"+k) != "" {
		return 1
	}
	return 0
}

func fakeKey(args []string) string {
	return strings.NewReplacer(" ", "_", "-", "_").Replace(strings.Join(args, " "))
}

// useFakeDdev makes ddev subcommands run the test binary as a fake ddev.
func useFakeDdev(t *testing.T) {
	t.Helper()
	orig := ddevBinary
	t.Cleanup(func() { ddevBinary = orig })
	ddevBinary = func() (string, error) { return os.Executable() }
	t.Setenv(fakeDdevEnv, "1")
}

// fakeOutput sets what the fake ddev prints for args, and whether it fails.
func fakeOutput(t *testing.T, stdout, stderr string, fail bool, args ...string) {
	t.Helper()
	k := fakeKey(args)
	t.Setenv("TUI_TEST_OUT_"+k, stdout)
	t.Setenv("TUI_TEST_ERR_"+k, stderr)
	if fail {
		t.Setenv("TUI_TEST_EXIT_"+k, "1")
	}
}

// newModel returns a model that received a window size, then projects when
// any are given, the way a running program would.
func newModel(t *testing.T, width, height int, projects ...ProjectInfo) AppModel {
	t.Helper()
	m := NewAppModel()
	m.dashboard.cwd = t.TempDir()
	m = send(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	if projects != nil {
		m = send(t, m, projectsLoadedMsg{projects: projects, router: health{status: "healthy"}})
	}
	return m
}

// send feeds msgs to m through Update, dropping the commands.
func send(t *testing.T, m AppModel, msgs ...tea.Msg) AppModel {
	t.Helper()
	for _, msg := range msgs {
		m, _ = sendCmd(t, m, msg)
	}
	return m
}

func sendCmd(t *testing.T, m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	model, ok := updated.(AppModel)
	require.True(t, ok)
	return model, cmd
}

// press returns the key press for a key name such as "enter" or "j".
func press(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	if letter, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: rune(letter[0]), Mod: tea.ModCtrl}
	}
	r, _ := utf8.DecodeRuneInString(k)
	return tea.KeyPressMsg{Code: r, Text: k}
}

// keys presses each key in turn.
func keys(t *testing.T, m AppModel, names ...string) AppModel {
	t.Helper()
	for _, k := range names {
		m = send(t, m, press(k))
	}
	return m
}

// typeText types s one character at a time, letting the list filter run.
func typeText(t *testing.T, m AppModel, s string) AppModel {
	t.Helper()
	for _, r := range s {
		var cmd tea.Cmd
		m, cmd = sendCmd(t, m, press(string(r)))
		m = settle(t, m, cmd)
	}
	return m
}

// settle runs cmd and feeds back the messages of the commands that finish
// at once, as the program would. Timers, such as the refresh tick and the
// spinner, are dropped.
func settle(t *testing.T, m AppModel, cmd tea.Cmd) AppModel {
	t.Helper()
	round := []tea.Cmd{cmd}
	for n := 0; len(round) > 0 && n < 20; n++ {
		results := make(chan tea.Msg, len(round))
		started := 0
		for _, c := range round {
			if c != nil {
				started++
				go func() { results <- c() }()
			}
		}
		var msgs []tea.Msg
		timeout := time.After(20 * time.Millisecond)
	collect:
		for range started {
			select {
			case msg := <-results:
				msgs = append(msgs, msg)
			case <-timeout:
				break collect
			}
		}
		round = nil
		for len(msgs) > 0 {
			msg := msgs[0]
			msgs = msgs[1:]
			switch msg := msg.(type) {
			case nil:
			case tea.BatchMsg:
				round = append(round, msg...)
			default:
				var next tea.Cmd
				m, next = sendCmd(t, m, msg)
				round = append(round, next)
			}
		}
	}
	return m
}

// screen returns the rendered view without styling.
func screen(m AppModel) string {
	return ansi.Strip(m.View().Content)
}

// headerLine is the first line of the screen, without its padding.
func headerLine(m AppModel) string {
	line, _, _ := strings.Cut(screen(m), "\n")
	return strings.TrimRight(line, " ")
}

// requireFits fails unless the view fills the window exactly, with no line
// wider than it.
func requireFits(t *testing.T, m AppModel, context string) {
	t.Helper()
	lines := strings.Split(m.View().Content, "\n")
	require.Len(t, lines, m.height, "%s: view height\n%s", context, screen(m))
	for i, line := range lines {
		require.LessOrEqual(t, ansi.StringWidth(line), m.width, "%s: line %d is too wide: %q", context, i, ansi.Strip(line))
	}
}

// sampleProjects returns a running, a stopped and a paused project.
func sampleProjects() []ProjectInfo {
	return []ProjectInfo{
		{Name: "alpha", Status: ddevapp.SiteRunning, Type: "drupal11", URL: "https://alpha.ddev.site", AppRoot: "/home/user/apps/alpha"},
		{Name: "beta", Status: ddevapp.SiteStopped, Type: "wordpress", AppRoot: "/home/user/apps/beta"},
		{Name: "gamma", Status: ddevapp.SitePaused, Type: "laravel", AppRoot: "/home/user/work/gamma"},
	}
}

// sampleDetail returns the detail of a running project.
func sampleDetail() ProjectDetail {
	return ProjectDetail{
		Name: "alpha", Status: ddevapp.SiteRunning, Type: "drupal11", AppRoot: "/home/user/apps/alpha",
		PHPVersion: "8.4", WebserverType: "nginx-fpm", NodeJSVersion: "24", Docroot: "web",
		DatabaseType: "mariadb", DatabaseVersion: "11.8", DBHostPort: "127.0.0.1:32768", DBInternal: "db:3306", DBShared: "ddev-alpha-db:3306", DockerHost: "127.0.0.1",
		PrimaryURL: "https://alpha.ddev.site",
		URLs:       []string{"https://alpha.ddev.site", "http://alpha.ddev.site"},
		MailpitURL: "https://alpha.ddev.site:8026",
		Services: []ServiceInfo{
			{Name: "web", Status: ddevapp.SiteRunning, URL: "https://alpha.ddev.site", Ports: []PortInfo{{"80", "32770"}, {"443", "32771"}, {"8025", ""}}},
			{Name: "db", Status: ddevapp.SiteRunning, Ports: []PortInfo{{"3306", "32768"}}},
			{Name: "redis", Status: ddevapp.SiteRunning, Ports: []PortInfo{{"6379", ""}}, Info: "Redis 7"},
		},
		Addons: []AddonInfo{{Name: "redis", Version: "v2.0.0", Repository: "ddev/ddev-redis"}},
		Snapshots: []ddevapp.SnapshotRestoreTarget{
			{Name: "alpha_20261001", Label: "alpha_20261001", Created: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)},
			{Name: "alpha_20260901", Label: "alpha_20260901", Created: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)},
		},
	}
}

// detailModel returns a model showing the loaded detail d.
func detailModel(t *testing.T, width, height int, d ProjectDetail) AppModel {
	t.Helper()
	m := newModel(t, width, height, sampleProjects()...)
	m, _ = m.openDetail(ProjectInfo{Name: d.Name, AppRoot: d.AppRoot, Status: d.Status})
	return send(t, m, projectDetailLoadedMsg{appRoot: d.AppRoot, detail: d})
}
