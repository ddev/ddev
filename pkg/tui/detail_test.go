package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

const (
	osc8Open  = "\x1b]8;;"
	osc8Close = "\x1b]8;;\x1b\\"
)

// requireBalancedLinks fails if s has an unclosed OSC 8 hyperlink.
func requireBalancedLinks(t *testing.T, s string) {
	t.Helper()
	opens := strings.Count(s, osc8Open) - strings.Count(s, osc8Close)
	require.Equal(t, opens, strings.Count(s, osc8Close), "every hyperlink needs a terminator: %q", s)
}

func TestOpenDetail(t *testing.T) {
	m := keys(t, newModel(t, 120, 30, sampleProjects()...), "j", "enter")
	require.Equal(t, viewDetail, m.view)
	require.Equal(t, "/home/user/apps/beta", m.detail.root)
	require.True(t, m.detail.loading)
	require.Contains(t, screen(m), "Loading project details...")

	require.Equal(t, viewDashboard, keys(t, newModel(t, 120, 30, []ProjectInfo{}...), "enter").view)

	for _, k := range []string{"esc", "backspace"} {
		back := keys(t, detailModel(t, 120, 30, sampleDetail()), k)
		require.Equal(t, viewDashboard, back.view, k)
		require.Nil(t, back.detail.project)
	}
}

// TestDetailNeverStartsProject checks that showing a stopped project runs no
// command in its containers, which would start it. The browser keys do start
// it, as `ddev launch` does, but only when pressed.
func TestDetailNeverStartsProject(t *testing.T) {
	c := stubCommands(t)
	stopped := sampleDetail()
	stopped.Name, stopped.AppRoot, stopped.Status = "beta", "/home/user/apps/beta", ddevapp.SiteStopped

	m := keys(t, newModel(t, 120, 30, sampleProjects()...), "j", "enter")
	m = send(t, m, projectDetailLoadedMsg{appRoot: stopped.AppRoot, detail: stopped})
	m = keys(t, m, "r")
	m = send(t, m, projectDetailLoadedMsg{appRoot: stopped.AppRoot, detail: stopped})
	keys(t, m, "X", "e", "L")
	require.Empty(t, c.xdebug)
	require.Empty(t, c.execs)
	require.Empty(t, c.streams)
	require.Empty(t, c.opens)

	// A running project's Xdebug state is read once its detail is loaded.
	send(t, keys(t, newModel(t, 120, 30, sampleProjects()...), "enter"), projectDetailLoadedMsg{appRoot: "/home/user/apps/alpha", detail: sampleDetail()})
	require.Equal(t, []string{"/home/user/apps/alpha status"}, c.xdebug)
}

// row is a labeled line of the detail view.
func row(label, value string) string {
	return " " + label + strings.Repeat(" ", labelWidth-len(label)) + value
}

func TestDetailView(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	m := detailModel(t, 120, 50, sampleDetail())
	m = send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha", enabled: true})
	view := screen(m)
	// The right column starts 4 cells after the widest left one, Webserver.
	pair := func(left, right string) string {
		return left + strings.Repeat(" ", ansi.StringWidth(row("Webserver", "nginx-fpm"))+4-ansi.StringWidth(left)) + right
	}
	for _, want := range []string{
		"alpha ", "running",
		pair(row("Type", "drupal11"), row("Docroot", "web")),
		pair(row("PHP", "8.4"), row("Perf mode", "none")),
		pair(row("Webserver", "nginx-fpm"), row("Xdebug", "on")),
		pair(row("Node.js", "24"), row("XHProf mode", "–")),
		row("Database", "mariadb 11.8 · db/db or root/root"),
		row("", "127.0.0.1:32768 from host"),
		row("", "db:3306 from this project · ddev-alpha-db:3306 from other projects"),
		row("Snapshots", "2 · latest alpha_20261001 · 2026-10-01"),
		row("Path", "/home/user/apps/alpha"), row("Open in", "VS Code · PhpStorm"),
		row("URLs", "https://alpha.ddev.site · http://alpha.ddev.site"),
		row("Mailpit", "https://alpha.ddev.site:8026"),
		"Ports (container→host)",
		" web      ● running  https://alpha.ddev.site  80→127.0.0.1:32770, 443→127.0.0.1:32771, 8025 (containers only)",
		" redis    ● running                           6379 (containers only)",
		"3306→127.0.0.1:32768",
		"6379 (containers only)                                           Redis 7",
		"Add-ons · find more at https://addons.ddev.com",
		"ddev/ddev-redis  v2.0.0",
	} {
		require.Contains(t, view, want)
	}

	content := m.detail.content(m.styles, m.detail.vp.Width())
	requireBalancedLinks(t, content)
	require.Contains(t, content, osc8Open+"file:///home/user/apps/alpha\x1b\\")
	require.Contains(t, content, osc8Open+"https://alpha.ddev.site:8026\x1b\\")
	require.Contains(t, content, osc8Open+"vscode://file/home/user/apps/alpha?windowId=_blank\x1b\\")
	require.Contains(t, content, osc8Open+"phpstorm://open?file=/home/user/apps/alpha\x1b\\")
	require.Contains(t, content, osc8Open+"https://github.com/ddev/ddev-redis\x1b\\")

	// Truncated links keep their full target and stay terminated.
	narrow := send(t, m, tea.WindowSizeMsg{Width: 30, Height: 50})
	requireBalancedLinks(t, narrow.View().Content)
	require.Contains(t, narrow.View().Content, osc8Open+"https://alpha.ddev.site:8026\x1b\\")

	stopped := sampleDetail()
	stopped.Status, stopped.DBHostPort = ddevapp.SiteStopped, ""
	view = screen(detailModel(t, 120, 50, stopped))
	require.Contains(t, view, row("", "no host port while stopped"))
	require.Contains(t, view, row("Xdebug", "off"), "a stopped project is not asked, since that would start it")

	noDB := sampleDetail()
	noDB.DBOmitted = true
	view = screen(detailModel(t, 120, 50, noDB))
	require.Contains(t, view, row("Database", "none"))
	require.NotContains(t, view, "db/db", "no credentials for a database that is not there")
	require.NotContains(t, view, "Snapshots")

	require.Contains(t, screen(detailModel(t, 40, 50, sampleDetail())), row("Snapshots", "2 · 2026-10-01"),
		"the name is cut before the date")

	noSnapshots := sampleDetail()
	noSnapshots.Snapshots = nil
	require.Contains(t, screen(detailModel(t, 120, 50, noSnapshots)), row("Snapshots", "none"))

	mutagen := sampleDetail()
	mutagen.PerformanceMode, mutagen.MutagenStatus, mutagen.XHGuiURL = "mutagen", "ok", "https://alpha.ddev.site:8142"
	view = screen(detailModel(t, 120, 50, mutagen))
	require.Contains(t, view, row("Perf mode", "mutagen (ok)"))
	require.Contains(t, view, row("XHGui", "https://alpha.ddev.site:8142"))

	noServices := sampleDetail()
	noServices.Status, noServices.Services = ddevapp.SiteStopped, nil
	require.Contains(t, screen(detailModel(t, 120, 50, noServices)), " None running")

	none := sampleDetail()
	none.Addons = nil
	view = screen(detailModel(t, 120, 50, none))
	require.Contains(t, view, "Add-ons · find more at https://addons.ddev.com")
	require.Contains(t, view, " None installed")
}

func TestDetailXdebugKeepsItsPlace(t *testing.T) {
	m := detailModel(t, 120, 50, sampleDetail())
	before := strings.Split(screen(m), "\n")
	require.Contains(t, screen(m), row("Xdebug", "…"), "a placeholder holds the place of the Xdebug state")

	after := strings.Split(screen(send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha"})), "\n")
	require.Len(t, after, len(before))
	changed := 0
	for i := range before {
		if before[i] != after[i] {
			changed++
			require.Contains(t, after[i], row("Xdebug", "off"))
		}
	}
	require.Equal(t, 1, changed, "only the Xdebug line changes when its state arrives")
}

func TestDetailServicesWhenNarrow(t *testing.T) {
	m := detailModel(t, 50, 60, sampleDetail())
	var trimmed []string
	for line := range strings.SplitSeq(screen(m), "\n") {
		trimmed = append(trimmed, strings.TrimRight(line, " "))
	}
	view := strings.Join(trimmed, "\n")
	require.Contains(t, view, " Ports (container→host)")
	require.Contains(t, view, " web      ● running\n          https://alpha.ddev.site\n          80→127.0.0.1:32770\n          443→127.0.0.1:32771\n          8025 (containers only)")
	require.Contains(t, view, " redis    ● running\n          6379 (containers only)\n          Redis 7")
	require.Contains(t, view, row("URLs", "https://alpha.ddev.site\n")+row("", "http://alpha.ddev.site\n"), "URLs wrap when they do not fit side by side")
	requireFits(t, m, "narrow detail")

	// Wide enough for the table, but not for its info column.
	view = screen(detailModel(t, 112, 60, sampleDetail()))
	require.Regexp(t, `\n redis +● running +6379 \(containers only\) *│?\n {10}Redis 7 `, view)

	noPorts := sampleDetail()
	for i := range noPorts.Services {
		noPorts.Services[i].Ports = nil
	}
	require.NotContains(t, screen(detailModel(t, 50, 60, noPorts)), "Ports", "no legend without a port to explain")
	require.NotContains(t, screen(detailModel(t, 120, 60, noPorts)), "Ports")
}

func TestDetailSharedContainerHealth(t *testing.T) {
	d := sampleDetail()
	shared := func(m AppModel, router, sshAgent health) AppModel {
		return send(t, m, projectsLoadedMsg{projects: sampleProjects(), router: router, sshAgent: sshAgent})
	}
	m := shared(detailModel(t, 120, 50, d), health{"healthy", "previously healthy, sleeping"}, health{"healthy", "ok"})
	header := headerLine(m)
	require.Contains(t, header, "alpha  ● running")
	require.True(t, strings.HasSuffix(header, "SSH agent ● healthy · Router ● healthy"), "on the right, as on the dashboard: %q", header)
	require.NotContains(t, screen(m), "previously healthy")

	m = shared(detailModel(t, 60, 40, d), health{"unhealthy", "traefik: " + longText}, health{ddevapp.SiteStopped, ""})
	view := screen(m)
	require.Contains(t, view, "SSH agent ○ stopped · Router ✗ unhealthy")
	require.Contains(t, view, "Router: traefik: very-long-segment")
	requireFits(t, m, "detail errors")
	require.Contains(t, screen(keys(t, m, "esc")), "Router: traefik: very-long-segment", "the dashboard shows the same")
}

func TestDetailLoads(t *testing.T) {
	m := newModel(t, 120, 30, sampleProjects()...)
	m, _ = m.openDetail(sampleProjects()[0])

	// A result for a project no longer shown is dropped.
	stale := sampleDetail()
	stale.Name, stale.AppRoot = "beta", "/home/user/apps/beta"
	m = send(t, m, projectDetailLoadedMsg{appRoot: stale.AppRoot, detail: stale}, xdebugStatusMsg{appRoot: stale.AppRoot, enabled: true})
	require.Nil(t, m.detail.project)
	require.Nil(t, m.detail.xdebug)
	require.True(t, m.detail.loading)

	m = send(t, m, projectDetailLoadedMsg{appRoot: "/home/user/apps/alpha", detail: sampleDetail(), warning: "Could not reach the router"})
	require.False(t, m.detail.loading)
	require.Equal(t, "alpha", m.detail.project.Name)
	require.Contains(t, screen(m), "Could not reach the router", "what ddevapp logged while loading shows in the status line")

	// A failed first load goes back to the dashboard with the error.
	failed, _ := newModel(t, 120, 30, sampleProjects()...).openDetail(sampleProjects()[1])
	failed = send(t, failed, projectDetailLoadedMsg{appRoot: "/home/user/apps/beta", err: errors.New("describe failed")})
	require.Equal(t, viewDashboard, failed.view)
	require.Contains(t, screen(failed), "Error loading beta: describe failed")

	// A failed reload keeps the detail it had.
	reloaded := send(t, m, projectDetailLoadedMsg{appRoot: "/home/user/apps/alpha", err: errors.New("boom")})
	require.Equal(t, viewDetail, reloaded.view)
	require.NotNil(t, reloaded.detail.project)
}

// TestDetailKeepsRows checks that the rows above the URLs stay where they are
// while the detail loads and when the project stops.
func TestDetailKeepsRows(t *testing.T) {
	lineOf := func(view, prefix string) int {
		for i, l := range strings.Split(view, "\n") {
			if strings.HasPrefix(l, prefix) {
				return i
			}
		}
		require.Failf(t, "row missing", "%q is not in\n%s", prefix, view)
		return -1
	}
	running := sampleDetail()
	fromConfig := sampleDetail()
	fromConfig.FromConfig, fromConfig.DBHostPort, fromConfig.URLs, fromConfig.MailpitURL = true, "", nil, ""
	stopped := sampleDetail()
	stopped.Status, stopped.DBHostPort = ddevapp.SiteStopped, ""

	want := screen(detailModel(t, 80, 50, running))
	for name, d := range map[string]ProjectDetail{"loading": fromConfig, "stopped": stopped} {
		view := screen(detailModel(t, 80, 50, d))
		for _, prefix := range []string{row("Database", ""), row("Path", ""), row("URLs", "")} {
			require.Equal(t, lineOf(want, prefix), lineOf(view, prefix), "%s: %q moved", name, prefix)
		}
	}
	require.Contains(t, screen(detailModel(t, 80, 50, fromConfig)), row("", "loading from host"))
	require.Contains(t, screen(detailModel(t, 80, 50, fromConfig)), row("Snapshots", "loading"))
}

func TestDetailAddonUpdates(t *testing.T) {
	var checked []string
	orig := latestAddons
	t.Cleanup(func() { latestAddons = orig })
	latestAddons = func(appRoot string) tea.Cmd {
		checked = append(checked, appRoot)
		return nil
	}

	m := detailModel(t, 120, 50, sampleDetail())
	require.Equal(t, []string{"/home/user/apps/alpha"}, checked, "the add-ons are checked once the detail loads")
	require.Regexp(t, `ddev/ddev-redis  v2\.0\.0 *\n`, screen(m))

	updated := send(t, m, latestAddonsMsg{appRoot: "/home/user/apps/alpha", versions: map[string]string{"ddev/ddev-redis": "v2.1.0"}})
	require.Contains(t, screen(updated), "ddev/ddev-redis  v2.0.0 → v2.1.0")
	require.True(t, updated.detail.keys(updated.keys).Update.Enabled())

	current := send(t, m, latestAddonsMsg{appRoot: "/home/user/apps/alpha", versions: map[string]string{"ddev/ddev-redis": "v2.0.0"}})
	require.NotContains(t, screen(current), "v2.0.0 →")
	require.False(t, current.detail.keys(current.keys).Update.Enabled(), "nothing to update")

	other := send(t, m, latestAddonsMsg{appRoot: "/home/user/apps/beta", versions: map[string]string{"ddev/ddev-redis": "v2.1.0"}})
	require.NotContains(t, screen(other), "v2.0.0 →", "the versions of another project's detail are dropped")

	stale := send(t, m, latestAddonsMsg{appRoot: "/home/user/apps/alpha", versions: map[string]string{"ddev/ddev-redis": "v1.9.0"}})
	require.NotContains(t, screen(stale), "v2.0.0 →", "a registry cached before the installed release offers no update")
	require.False(t, stale.detail.keys(stale.keys).Update.Enabled())

	branch := sampleDetail()
	branch.Addons[0].Version = "main"
	onBranch := send(t, detailModel(t, 120, 50, branch), latestAddonsMsg{appRoot: "/home/user/apps/alpha", versions: map[string]string{"ddev/ddev-redis": "v2.1.0"}})
	require.Contains(t, screen(onBranch), "ddev/ddev-redis  main → v2.1.0", "add-on update replaces a branch with the latest release")

	noServices := sampleDetail()
	noServices.Status, noServices.Services = ddevapp.SiteStopped, nil
	require.Contains(t, screen(detailModel(t, 120, 50, noServices)), " None running")

	none := sampleDetail()
	none.Addons = nil
	checked = nil
	detailModel(t, 120, 50, none)
	require.Empty(t, checked, "no add-ons, nothing to check")
}

func TestDetailStartsFromList(t *testing.T) {
	c := stubCommands(t)
	alpha := sampleProjects()[0]
	alpha.Detail = &ProjectDetail{Name: alpha.Name, Status: statusLoading, Type: alpha.Type, AppRoot: alpha.AppRoot, PHPVersion: "8.4", FromConfig: true}
	m, _ := newModel(t, 120, 30, sampleProjects()...).openDetail(alpha)
	require.True(t, m.detail.loading)
	view := screen(send(t, m, tea.WindowSizeMsg{Width: 120, Height: 30}))
	require.Contains(t, view, "alpha  ● running", "the status the list has shows at once")
	require.Contains(t, view, row("URLs", alpha.URL), "and so does its URL")
	keys(t, m, "e")
	require.Equal(t, []string{"/home/user/apps/alpha ssh"}, c.execs, "so do the keys of a running project")
	require.True(t, c.ops[0].shell, "its exit status is the user's last command")
}

func TestDetailRefresh(t *testing.T) {
	m := detailModel(t, 120, 30, sampleDetail())
	m = send(t, m, tickMsg{})
	require.True(t, m.dashboard.loading, "the tick refreshes the project list")
	require.False(t, m.detail.loading, "but not the detail, whose Describe inspects every container")

	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.False(t, m.detail.loading, "alpha is still running")

	changed := sampleProjects()
	changed[0].Status = ddevapp.SiteStopped
	m = send(t, m, projectsLoadedMsg{projects: changed})
	require.True(t, m.detail.loading, "alpha stopped elsewhere, so its detail is reloaded")

	gone := send(t, detailModel(t, 120, 30, sampleDetail()), projectsLoadedMsg{projects: sampleProjects()[1:]})
	require.False(t, gone.detail.loading, "a project no longer listed keeps the detail it had")

	r := keys(t, detailModel(t, 120, 30, sampleDetail()), "r")
	require.True(t, r.detail.loading)
	require.True(t, r.refreshing)
}

func TestDetailKeys(t *testing.T) {
	c := stubCommands(t)
	require.Empty(t, keys(t, detailModel(t, 120, 30, sampleDetail()), "x").stream.name, "not in xhgui mode")
	d := sampleDetail()
	d.XHProfMode = types.XHProfModeXHGui
	m := detailModel(t, 120, 30, d)
	for _, k := range []string{"s", "S", "R"} {
		keys(t, m, k)
	}
	for _, k := range []string{"l", "m", "x", "e"} {
		keys(t, m, k)
	}
	require.Equal(t, []string{
		"/home/user/apps/alpha start", "/home/user/apps/alpha stop", "/home/user/apps/alpha restart",
		"/home/user/apps/alpha launch", "/home/user/apps/alpha mailpit", "/home/user/apps/alpha xhgui",
		"/home/user/apps/alpha ssh",
	}, c.execs)
	require.Empty(t, c.streams)

	logs := keys(t, m, "L")
	require.Equal(t, []string{"/home/user/apps/alpha logs -f"}, c.streams)
	require.Equal(t, "Logs: alpha", logs.stream.name)
	require.Equal(t, viewDetail, logs.stream.returnView)

	require.Equal(t, viewDetail, keys(t, m, "s").view, "the detail stays behind the command")

	// Keys that need running containers do nothing for a stopped project.
	stopped := sampleDetail()
	stopped.Status = ddevapp.SiteStopped
	c.execs, c.streams = nil, nil
	s := detailModel(t, 120, 30, stopped)
	for _, k := range []string{"e", "L", "X"} {
		s = keys(t, s, k)
	}
	require.Empty(t, c.execs)
	require.Empty(t, c.streams)
	require.Equal(t, viewDetail, s.view)
	require.NotContains(t, screen(s), "Toggling Xdebug")
}

func TestXdebug(t *testing.T) {
	c := stubCommands(t)
	m := detailModel(t, 120, 30, sampleDetail())
	require.Nil(t, keys(t, m, "X").confirm)
	require.NotContains(t, screen(keys(t, m, "X")), "Toggling Xdebug", "nothing to toggle until the state is known")

	m = send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha"})
	c.xdebug = nil
	m, cmd := sendCmd(t, m, press("X"))
	require.NotNil(t, cmd)
	require.Contains(t, screen(m), "Toggling Xdebug...")
	settle(t, m, cmd)
	require.Equal(t, []string{"/home/user/apps/alpha on"}, c.xdebug, "the known state says which way to switch")
	_, cmd = sendCmd(t, m, press("X"))
	settle(t, m, cmd)
	require.Len(t, c.xdebug, 1, "a second press waits for the reply")

	on := send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha", enabled: true, toggled: true})
	require.Contains(t, screen(on), "Xdebug enabled")
	require.True(t, *on.detail.xdebug)
	c.xdebug = nil
	settle(t, on, nil)
	_, cmd = sendCmd(t, on, press("X"))
	settle(t, on, cmd)
	require.Equal(t, []string{"/home/user/apps/alpha off"}, c.xdebug)

	// It is read once: a reload for the 5-second refresh keeps it, and a
	// change of status forgets it.
	c.xdebug = nil
	same := send(t, on, projectDetailLoadedMsg{appRoot: "/home/user/apps/alpha", detail: sampleDetail()})
	require.True(t, *same.detail.xdebug)
	require.Empty(t, c.xdebug)
	stopped := sampleDetail()
	stopped.Status = ddevapp.SiteStopped
	require.Nil(t, send(t, on, projectDetailLoadedMsg{appRoot: "/home/user/apps/alpha", detail: stopped}).detail.xdebug)

	off := send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha", toggled: true})
	require.Contains(t, screen(off), "Xdebug disabled")

	failed := send(t, m, xdebugStatusMsg{appRoot: "/home/user/apps/alpha", toggled: true, err: errors.New("exit 1")})
	require.Contains(t, screen(failed), "Xdebug toggle failed: exit 1")

	// A failed status check alone is not worth a message.
	quiet := send(t, detailModel(t, 120, 30, sampleDetail()), xdebugStatusMsg{appRoot: "/home/user/apps/alpha", err: errors.New("exit 1")})
	require.NotContains(t, screen(quiet), "exit 1")
	require.Nil(t, quiet.detail.xdebug)
}

func TestCopyURL(t *testing.T) {
	var copied string
	orig := writeClipboard
	t.Cleanup(func() { writeClipboard = orig })
	writeClipboard = func(s string) error {
		copied = s
		return nil
	}
	m := detailModel(t, 120, 30, sampleDetail())
	_, cmd := sendCmd(t, m, press("c"))
	msg := cmd()
	require.Equal(t, "https://alpha.ddev.site", copied)
	require.Contains(t, screen(send(t, m, msg)), "Copied the URL to the clipboard")

	writeClipboard = func(string) error { return errors.New("no clipboard tool") }
	_, cmd = sendCmd(t, m, press("c"))
	msg = cmd()
	require.Equal(t, clipboardMsg{fallback: "https://alpha.ddev.site"}, msg)
	m, cmd = sendCmd(t, m, msg)
	require.NotNil(t, cmd, "the terminal's clipboard is asked instead")
	require.Contains(t, screen(m), "Sent the URL to the terminal's clipboard")

	noURL := sampleDetail()
	noURL.PrimaryURL = ""
	_, cmd = sendCmd(t, detailModel(t, 120, 30, noURL), press("c"))
	require.Nil(t, cmd)
}

// scrollbar is the last column of the detail view's body.
func scrollbar(m AppModel) string {
	lines := strings.Split(screen(m), "\n")[2:]
	var bar []rune
	for _, line := range lines[:m.detail.vp.Height()] {
		r := []rune(line)
		bar = append(bar, r[len(r)-1])
	}
	return string(bar)
}

func TestDetailScroll(t *testing.T) {
	m := detailModel(t, 80, 12, busyDetail())
	require.True(t, m.detail.vp.AtTop())
	require.Regexp(t, `^┃+│+$`, scrollbar(m))
	m = keys(t, m, "pgdown")
	require.False(t, m.detail.vp.AtTop())
	m = keys(t, m, "G")
	require.True(t, m.detail.vp.AtBottom())
	require.Regexp(t, `^│+┃+$`, scrollbar(m))
	requireFits(t, m, "scrolled detail")
	require.Equal(t, strings.Repeat(" ", 30), scrollbar(detailModel(t, 200, 40, sampleDetail()))[:30], "no bar when everything fits")
	require.True(t, keys(t, m, "g").detail.vp.AtTop())
	require.False(t, keys(t, m, "b").detail.vp.AtBottom())
	for _, k := range []string{"u", "ctrl+u"} {
		require.False(t, keys(t, m, k).detail.vp.AtBottom(), "%s scrolls half a page up", k)
		require.True(t, keys(t, m, k, "d").detail.vp.AtBottom(), "d scrolls back down")
	}
	require.True(t, keys(t, m, "g", "space").detail.vp.AtTop(), "space does not scroll")
}

func TestEditorURL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	require.Equal(t, "vscode://file/home/user/my%20site?windowId=_blank", editorURL("vscode", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=/home/user/my%20site", editorURL("phpstorm", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=/home/user/a%26b%2Bc", editorURL("phpstorm", "/home/user/a&b+c"))
	require.Equal(t, "vscode://file/C:/Users/me/site?windowId=_blank", editorURL("vscode", "C:/Users/me/site"))
	require.Empty(t, editorURL("unknown", "/home/user"))

	t.Setenv("WSL_DISTRO_NAME", "Ubuntu-26.04")
	require.Equal(t, "vscode://vscode-remote/wsl+Ubuntu-26.04/home/user/my%20site?windowId=_blank", editorURL("vscode", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=//wsl.localhost/Ubuntu-26.04/home/user/my%20site", editorURL("phpstorm", "/home/user/my site"))
}

func TestDashboardHyperlinks(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	out := m.View().Content
	require.Contains(t, out, osc8Open+"file:///home/user/apps/alpha\x1b\\")
	require.Contains(t, out, osc8Open+"https://alpha.ddev.site\x1b\\")
	requireBalancedLinks(t, out)

	narrow := send(t, m, tea.WindowSizeMsg{Width: 50, Height: 24})
	requireBalancedLinks(t, narrow.View().Content)
	require.Contains(t, narrow.View().Content, osc8Open+"file:///home/user/work/gamma\x1b\\", "the full path stays the link target when the text is cut")
}

func TestDetailSnapshotKeys(t *testing.T) {
	c := stubCommands(t)
	m := detailModel(t, 120, 50, sampleDetail())
	keys(t, m, "n")
	require.Equal(t, []string{"/home/user/apps/alpha snapshot"}, c.execs)

	restore := keys(t, m, "N")
	require.Contains(t, screen(restore), "Replace the database of 'alpha' with snapshot alpha_20261001 from 2026-10-01?")
	require.Nil(t, keys(t, restore, "n").confirm, "any other key cancels")
	require.Len(t, c.execs, 1)
	require.Equal(t, viewDetail, keys(t, restore, "y").view)
	require.Equal(t, "/home/user/apps/alpha snapshot restore alpha_20261001", c.execs[1])

	// A snapshot would start a stopped project, which a restore asks first.
	c.execs = nil
	stopped := sampleDetail()
	stopped.Status = ddevapp.SiteStopped
	m = keys(t, detailModel(t, 120, 50, stopped), "n")
	require.Empty(t, c.execs)
	require.NotNil(t, keys(t, m, "N").confirm)

	noDB := sampleDetail()
	noDB.DBOmitted = true
	m = keys(t, detailModel(t, 120, 50, noDB), "n", "N")
	require.Empty(t, c.execs)
	require.Nil(t, m.confirm)

	none := sampleDetail()
	none.Snapshots = nil
	require.Nil(t, keys(t, detailModel(t, 120, 50, none), "N").confirm, "nothing to restore")

	worktree := sampleDetail()
	worktree.Snapshots[0].Label = "alpha_20261001 (alpha-feature)"
	m = detailModel(t, 120, 50, worktree)
	require.Contains(t, screen(m), row("Snapshots", "2 · latest alpha_20261001 (alpha-feature) · 2026-10-01"))
	require.Contains(t, screen(keys(t, m, "N")), "with snapshot alpha_20261001 (alpha-feature) from 2026-10-01?")
}

func TestDetailUpdateKey(t *testing.T) {
	c := stubCommands(t)
	m := detailModel(t, 120, 50, sampleDetail())
	require.Nil(t, keys(t, m, "U").confirm, "no newer release known")

	m = send(t, m, latestAddonsMsg{appRoot: "/home/user/apps/alpha", versions: map[string]string{"ddev/ddev-redis": "v2.1.0"}})
	m = keys(t, m, "U")
	require.Contains(t, screen(m), "Update the add-ons of 'alpha'? ddev/ddev-redis v2.0.0 → v2.1.0")
	m = keys(t, m, "y")
	require.Equal(t, []string{"/home/user/apps/alpha add-on update --yes ddev/ddev-redis"}, c.execs, "only the listed add-ons update")

	m = send(t, m, execFinishedMsg{title: c.ops[0].title, done: c.ops[0].done})
	require.Equal(t, viewDetail, m.view)
	require.Contains(t, screen(m), "Updated the add-ons of alpha, restart with R to use them")
}
