package tui

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/stretchr/testify/require"
)

// calls records the subprocesses and openers that keys asked for.
type calls struct {
	streams []string
	execs   []string
	ops     []operationAction
	opens   []string
	xdebug  []string
}

// stubCommands replaces what keys run with recorders.
func stubCommands(t *testing.T) *calls {
	t.Helper()
	c := &calls{}
	origStream, origExec, origXdebug := startStream, execDdev, xdebugStatus
	t.Cleanup(func() { startStream, execDdev, xdebugStatus = origStream, origExec, origXdebug })
	xdebugStatus = func(appRoot, arg string) tea.Cmd {
		c.xdebug = append(c.xdebug, appRoot+" "+arg)
		return nil
	}
	startStream = func(_ int, dir string, args ...string) tea.Cmd {
		c.streams = append(c.streams, strings.TrimSpace(dir+" "+strings.Join(args, " ")))
		return nil
	}
	execDdev = func(op operationAction) tea.Cmd {
		c.execs = append(c.execs, strings.TrimSpace(op.dir+" "+strings.Join(op.args, " ")))
		c.ops = append(c.ops, op)
		return nil
	}
	stubOpen(t)
	openTargetCmd = func(what, target string) tea.Cmd {
		c.opens = append(c.opens, what+"|"+target)
		return nil
	}
	openInPhpStormCmd = func(dir string) tea.Cmd {
		c.opens = append(c.opens, "PhpStorm|"+dir)
		return nil
	}
	return c
}

func stubOpen(t *testing.T) {
	t.Helper()
	origTarget, origPhpStorm := openTargetCmd, openInPhpStormCmd
	t.Cleanup(func() { openTargetCmd, openInPhpStormCmd = origTarget, origPhpStorm })
	openTargetCmd = func(string, string) tea.Cmd { return nil }
	openInPhpStormCmd = func(string) tea.Cmd { return nil }
}

func filterTo(t *testing.T, m AppModel, text string) AppModel {
	t.Helper()
	return keys(t, typeText(t, keys(t, m, "/"), text), "enter")
}

func visibleNames(m AppModel) []string {
	var names []string
	for _, it := range m.dashboard.visibleItems() {
		names = append(names, it.Name)
	}
	return names
}

func TestDashboardShowsProjects(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	view := screen(m)
	for _, want := range []string{"DDEV v", "alpha", "running", "drupal11", "beta", "stopped", "https://alpha.ddev.site", "3 projects · 1 running"} {
		require.Contains(t, view, want)
	}
	require.NotContains(t, view, "All projects are stopped")
	require.Contains(t, view, "> alpha  ● running    drupal11   /home/user/apps/alpha  https://alpha.ddev.site", "columns are two cells apart")
	require.Equal(t, "alpha", m.dashboard.selected().Name)

	narrow := screen(send(t, m, tea.WindowSizeMsg{Width: 70, Height: 24}))
	require.Contains(t, narrow, "/home/user/apps/alpha  https://alph…", "the path comes first, and the URL is cut")

	long := append(sampleProjects(), ProjectInfo{Name: strings.Repeat("x", 30), Status: ddevapp.SiteStopped, AppRoot: "/home/user/x"})
	view = screen(newModel(t, 120, 24, long...))
	require.Contains(t, view, strings.Repeat("x", 30)+"  ○ stopped", "a long name takes the room the URLs leave empty")
	view = screen(newModel(t, 80, 24, long...))
	require.Contains(t, view, strings.Repeat("x", 19)+"…", "a long name is cut when the URLs need the room")
	require.Contains(t, view, "> alpha"+strings.Repeat(" ", 15+columnGap)+"● running", "instead of widening the column for every project")
}

func TestDashboardStates(t *testing.T) {
	m := newModel(t, 80, 24)
	require.Contains(t, screen(m), "Loading projects...")

	failed := send(t, m, projectsLoadedMsg{err: errors.New("Docker error: cannot connect")})
	require.Contains(t, screen(failed), "Docker error: cannot connect")

	empty := send(t, m, projectsLoadedMsg{projects: []ProjectInfo{}})
	require.Contains(t, screen(empty), "No DDEV projects found.")
	require.Contains(t, screen(empty), "Press C to run ddev config")
	require.Empty(t, empty.dashboard.hint(empty.keys), "no hint to start projects that are not there")
	require.Equal(t, "? more • q quit", strings.TrimSpace(lastLine(screen(empty))), "and no keys that act on projects")

	one := send(t, m, projectsLoadedMsg{projects: sampleProjects()[1:2], warning: "Project 'gone' was removed"})
	require.Regexp(t, `(?m)^  1 project *$`, screen(one), "a single stopped project, none running")
	require.Contains(t, screen(one), "Project 'gone' was removed", "what ddevapp logged while loading shows in the status line")
	require.False(t, one.statusIsErr, "as a message that asks nothing of the user")
	require.Empty(t, send(t, one, statusExpiredMsg{id: one.statusID}).status, "and clears itself")
	failedLoad := send(t, newModel(t, 80, 24, sampleProjects()...), projectsLoadedMsg{err: errors.New("boom"), warning: "Project 'gone' was removed"})
	require.True(t, failedLoad.statusIsErr, "a failed load is still an error")

	// A failed refresh keeps the projects it had and reports the error.
	loaded := send(t, newModel(t, 80, 24, sampleProjects()...), projectsLoadedMsg{err: errors.New("boom")})
	require.Contains(t, screen(loaded), "alpha")
	require.Contains(t, screen(loaded), "Error loading projects: boom")
	recovered := send(t, loaded, projectsLoadedMsg{projects: sampleProjects()})
	require.Nil(t, recovered.dashboard.err)
}

func TestMutedColor(t *testing.T) {
	require.Equal(t, lipgloss.BrightBlack, mutedColor(nil, color.Black), "the theme's gray until both colors are known")

	// The muted color sits between the terminal's background and text, on
	// either kind of background.
	level := func(c color.Color) uint32 { r, _, _, _ := c.RGBA(); return r >> 8 }
	onDark := level(mutedColor(color.White, color.Black))
	require.True(t, onDark > 0x80 && onDark < 0xc0, "%#x", onDark)
	onLight := level(mutedColor(color.Black, color.White))
	require.True(t, onLight > 0x40 && onLight < 0x80, "%#x", onLight)

	m := newModel(t, 120, 24, sampleProjects()...)
	m = send(t, m, tea.ForegroundColorMsg{Color: color.Black}, tea.BackgroundColorMsg{Color: color.White})
	light := NewStyles(color.Black, color.White)
	require.Equal(t, light, m.styles)
	require.Equal(t, light.help(), m.help.Styles)
	require.Equal(t, light.filter(), m.dashboard.filter.Styles(), "the filter renders with them too")
}

func TestStatusColors(t *testing.T) {
	s := NewStyles(nil, nil)
	require.Equal(t, s.Running.Render("● running"), s.status(ddevapp.SiteRunning))
	require.Equal(t, s.Running.Render("● healthy"), s.status("healthy"))
	require.Equal(t, s.Paused.Render("◐ starting"), s.status(ddevapp.SiteStarting))
	require.Equal(t, s.Muted.Render("○ stopped"), s.status(ddevapp.SiteStopped), "stopped is normal, not an error")
	require.Equal(t, s.Muted.Render("⊘ disabled"), s.status(containerDisabled))
	for _, fault := range []string{ddevapp.SiteUnhealthy, "exited"} {
		require.Equal(t, s.Error.Render("✗ "+fault), s.status(fault))
	}
	require.Equal(t, s.Error.Render("✗ missing"), s.status(ddevapp.SiteDirMissing), "long statuses are shortened")
	require.Equal(t, s.Error.Render("✗ no config"), s.status(ddevapp.SiteConfigMissing))
	missing := []projectItem{{Name: "a", Status: ddevapp.SiteDirMissing}}
	require.Equal(t, projectColumns(nil, 120).status, projectColumns(missing, 120).status, "so the column does not widen")
	require.Equal(t, s.Muted.Render("◌ loading"), s.status(statusLoading))
	require.Empty(t, s.status(""))
}

func TestStoppedRouter(t *testing.T) {
	stopped := sampleProjects()
	for i := range stopped {
		stopped[i].Status = ddevapp.SiteStopped
	}
	router := func(projects []ProjectInfo, status string) string {
		m := newModel(t, 120, 24)
		return send(t, m, projectsLoadedMsg{projects: projects, router: health{status: status}}).health(false)
	}
	s := NewStyles(nil, nil)
	require.Contains(t, router(stopped, ddevapp.SiteStopped), s.status(ddevapp.SiteStopped), "DDEV removes the router when no project runs")
	require.Contains(t, router(sampleProjects(), ddevapp.SiteStopped), s.Error.Render("✗ stopped"), "running projects need the router")
	require.Contains(t, router(sampleProjects(), containerDisabled), s.status(containerDisabled), "unless it is disabled")
}

func TestSharedContainerHealth(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	load := func(router, ssh health) AppModel {
		return send(t, m, projectsLoadedMsg{projects: sampleProjects(), router: router, sshAgent: ssh})
	}

	ok := load(health{"healthy", "previously healthy, sleeping"}, health{"healthy", "ok"})
	header := headerLine(ok)
	require.True(t, strings.HasPrefix(header, "DDEV v"), header)
	require.True(t, strings.HasSuffix(header, "SSH agent ● healthy · Router ● healthy"), "on the right, as in the detail view: %q", header)
	require.NotContains(t, screen(ok), "previously healthy", "a healthy container's log is routine")
	require.Equal(t, lipgloss.Height(ok.footer()), 2, "a healthy footer is the divider and the help")

	stoppedAgent := screen(load(health{"healthy", ""}, health{ddevapp.SiteStopped, ""}))
	require.Contains(t, stoppedAgent, "SSH agent ○ stopped")

	compact := headerLine(send(t, ok, tea.WindowSizeMsg{Width: 60, Height: 24}))
	require.True(t, strings.HasSuffix(compact, "SSH ● · Router ●"), "narrow, only the symbols show: %q", compact)
	narrow := headerLine(send(t, ok, tea.WindowSizeMsg{Width: 40, Height: 24}))
	require.NotContains(t, narrow, "Router", "the status is left out when it does not fit")

	broken := load(health{"unhealthy", "traefik:\n  config error in project-a"}, health{"unhealthy", "socket missing"})
	view := screen(broken)
	require.Contains(t, view, "SSH agent ✗ unhealthy · Router ✗ unhealthy")
	require.Contains(t, view, "Router: traefik: config error in project-a")
	require.Contains(t, view, "SSH agent: socket missing")
	requireFits(t, broken, "health problems")

	// A long log takes at most two lines.
	long := load(health{"unhealthy", strings.Repeat("error ", 100)}, health{})
	var logLines []string
	for line := range strings.SplitSeq(screen(long), "\n") {
		if strings.Contains(line, "error error") {
			logLines = append(logLines, strings.TrimRight(line, " "))
		}
	}
	require.Len(t, logLines, 2)
	require.True(t, strings.HasSuffix(logLines[1], "…"), "the cut is marked: %q", logLines[1])
	requireFits(t, long, "long health log")
}

func TestAllStoppedHint(t *testing.T) {
	stopped := sampleProjects()
	for i := range stopped {
		stopped[i].Status, stopped[i].URL = ddevapp.SiteStopped, ""
	}
	m := newModel(t, 120, 24, stopped...)
	require.Contains(t, screen(m), "All projects are stopped. Press s to start the selected one, a to start all.")
	require.Regexp(t, `\n  3 projects *\n`, screen(m), "no running count when none run")

	// A refresh can leave an applied filter without matches.
	m = keys(t, typeText(t, keys(t, newModel(t, 120, 24, sampleProjects()...), "/"), "running"), "enter")
	m = send(t, m, projectsLoadedMsg{projects: stopped})
	require.Contains(t, screen(m), "No matching projects.")
	require.Contains(t, screen(m), "All projects are stopped. Press a to start all.", "with none selected")
}

func TestListScrolls(t *testing.T) {
	var projects []ProjectInfo
	for i := range 30 {
		projects = append(projects, ProjectInfo{Name: fmt.Sprintf("p%02d", i), Status: ddevapp.SiteStopped, Type: "php", AppRoot: fmt.Sprintf("/home/user/p%02d", i)})
	}
	m := newModel(t, 80, 16, projects...)
	rows := m.dashboard.height
	require.Contains(t, screen(m), "  30 projects")
	require.Regexp(t, `^┃+│+$`, strings.TrimSpace(listScrollbar(m)))

	// Moving past the last visible row scrolls by one row, not a page.
	for range rows {
		m = keys(t, m, "j")
	}
	view := screen(m)
	require.NotContains(t, view, "p00 ")
	require.Contains(t, view, "  p01 ")
	require.Contains(t, view, fmt.Sprintf("> p%02d ", rows))

	m = keys(t, m, "pgdown")
	require.Equal(t, 2*rows, m.dashboard.cursor, "pgdown moves the selection a screen down")
	require.Contains(t, screen(m), fmt.Sprintf("> p%02d ", 2*rows))
	m = keys(t, m, "G")
	require.Equal(t, 29, m.dashboard.cursor)
	require.Regexp(t, `^│+┃+$`, strings.TrimSpace(listScrollbar(m)))
	// A mouse wheel sends several arrows a notch, which overshoot the end.
	require.Equal(t, 29, keys(t, m, "j", "j", "j").dashboard.cursor, "the selection stops at the last project")
	require.Equal(t, 2, keys(t, newModel(t, 80, 16, sampleProjects()...), "pgdown").dashboard.cursor, "pgdown reaches the last project when all fit")
	require.Contains(t, screen(keys(t, m, "g")), "> p00 ", "the view follows the selection back up")
	m = keys(t, m, "pgup", "pgup", "pgup", "pgup")
	require.Equal(t, 0, m.dashboard.cursor)
	require.Contains(t, screen(m), "> p00 ")
	requireFits(t, m, "scrolled list")

	m = typeText(t, keys(t, m, "/"), "p1")
	require.Contains(t, screen(m), "  Filter: p1")
	require.Contains(t, screen(m), "12 of 30 projects")
}

// listScrollbar is the last column of the dashboard's list rows.
func listScrollbar(m AppModel) string {
	lines := strings.Split(screen(m), "\n")
	var bar []rune
	for _, line := range lines[3 : 3+m.dashboard.height] {
		r := []rune(line)
		bar = append(bar, r[len(r)-1])
	}
	return string(bar)
}

func TestStatusChangeKeepsColumns(t *testing.T) {
	projects := sampleProjects()
	before := screen(newModel(t, 120, 24, projects...))
	projects[1].Status = ddevapp.SiteStarting
	after := screen(newModel(t, 120, 24, projects...))
	column := func(view, name string) int {
		for line := range strings.SplitSeq(view, "\n") {
			if strings.Contains(line, " "+name+" ") {
				return strings.Index(line, "/home")
			}
		}
		return -1
	}
	require.Positive(t, column(before, "gamma"))
	require.Equal(t, column(before, "gamma"), column(after, "gamma"), "a longer status does not move the path column")
}

func TestLongNameColumn(t *testing.T) {
	name := "drupal-commerce-marketplace-staging"
	projects := []ProjectInfo{{Name: name, Status: ddevapp.SiteStopped, Type: "php", AppRoot: "/home/user/" + name}}
	require.Contains(t, screen(newModel(t, 200, 24, projects...)), "> "+name+" ", "a wide window shows the whole name")
	require.Contains(t, screen(newModel(t, 70, 24, projects...)), "> drupal-commerce-mar…", "a narrow one cuts it at 20")
}

func TestStatusChangeKeepsOrder(t *testing.T) {
	projects := []ProjectInfo{
		{Name: "a", Status: ddevapp.SiteStopped, AppRoot: "/a"},
		{Name: "b", Status: ddevapp.SiteRunning, AppRoot: "/b"},
		{Name: "c", Status: ddevapp.SitePaused, AppRoot: "/c"},
	}
	m := keys(t, newModel(t, 80, 24, projects...), "G")
	require.Equal(t, []string{"a", "b", "c"}, visibleNames(m), "the order ddev list gives")

	// A project started outside the TUI stays where it was.
	projects[2].Status = ddevapp.SiteRunning
	m = send(t, m, projectsLoadedMsg{projects: projects})
	require.Equal(t, []string{"a", "b", "c"}, visibleNames(m))
	require.Equal(t, "c", m.dashboard.selected().Name)
}

func TestNavigation(t *testing.T) {
	m := newModel(t, 80, 24, sampleProjects()...)
	require.Equal(t, "beta", keys(t, m, "j").dashboard.selected().Name)
	require.Equal(t, "gamma", keys(t, m, "down", "down", "down").dashboard.selected().Name, "the cursor stops at the last project")
	require.Equal(t, "alpha", keys(t, m, "j", "k", "up").dashboard.selected().Name, "and at the first")
	require.Equal(t, "gamma", keys(t, m, "G").dashboard.selected().Name)
	require.Equal(t, "alpha", keys(t, m, "G", "g").dashboard.selected().Name)
}

func TestFilter(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	for query, want := range map[string][]string{
		"beta":      {"beta"},
		"wordpress": {"beta"},
		"stopped":   {"beta"},
		"BETA":      {"beta"},
		"nomatch":   nil,
	} {
		f := typeText(t, keys(t, m, "/"), query)
		require.True(t, f.dashboard.filtering())
		require.Equal(t, want, visibleNames(f), "query %q", query)
	}

	// The directory name matches, but not the rest of the path, which most
	// queries would match.
	p := sampleProjects()
	p[2].AppRoot = "/home/user/work/gamma-checkout"
	m = newModel(t, 120, 24, p...)
	require.Equal(t, []string{"gamma"}, visibleNames(typeText(t, keys(t, m, "/"), "checkout")))
	require.Empty(t, visibleNames(typeText(t, keys(t, m, "/"), "home")))
}

func TestFilterLine(t *testing.T) {
	m := typeText(t, keys(t, newModel(t, 80, 24, sampleProjects()...), "/"), "al")
	typing := strings.Split(screen(m), "\n")
	require.True(t, strings.HasPrefix(typing[2], "  Filter: al"), "aligned with the names: %q", typing[2])
	require.True(t, strings.HasSuffix(strings.TrimRight(typing[2], " "), "2 of 3 projects"), typing[2])

	applied := strings.Split(screen(keys(t, m, "enter")), "\n")
	require.Equal(t, typing[3:5], applied[3:5], "applying the filter does not move the list")
	require.Contains(t, applied[2], "Filter: al")
}

func TestFilterRanksNameMatchesFirst(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	names := visibleNames(typeText(t, keys(t, m, "/"), "al"))
	require.Equal(t, "alpha", names[0], "fuzzy matching alone ranks gamma (laravel) first")
	require.Contains(t, names, "gamma")
}

func TestFilterReportsRuneOffsets(t *testing.T) {
	target := "alpha php running café-shop"
	ranks := filterProjects("shop", []string{target})
	require.Len(t, ranks, 1)
	at := len([]rune(target[:strings.Index(target, "shop")]))
	require.Equal(t, []int{at, at + 1, at + 2, at + 3}, ranks[0].MatchedIndexes, "lipgloss.StyleRunes counts runes, not bytes")
}

func TestFilterKeys(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	f := typeText(t, keys(t, m, "/"), "bet")
	require.Contains(t, screen(f), "Filter: bet")
	require.Contains(t, screen(f), "enter apply filter")

	applied := keys(t, f, "enter")
	require.False(t, applied.dashboard.filtering())
	require.Equal(t, "bet", applied.dashboard.filter.Value())
	require.Equal(t, "beta", applied.dashboard.selected().Name)
	require.Contains(t, screen(applied), "esc clear filter")

	cleared := keys(t, applied, "esc")
	require.Empty(t, cleared.dashboard.filter.Value())
	require.Len(t, visibleNames(cleared), 3)

	cancelled := keys(t, f, "esc")
	require.False(t, cancelled.dashboard.filtering())
	require.Len(t, visibleNames(cancelled), 3)

	// Action keys are typed into the filter rather than acting.
	c := stubCommands(t)
	typed := typeText(t, keys(t, m, "/"), "sSrDq")
	require.Equal(t, "sSrDq", typed.dashboard.filter.Value())
	require.Empty(t, c.streams)
	require.Nil(t, typed.confirm)

	for _, k := range []string{"tab", "up", "down"} {
		require.False(t, keys(t, f, k).dashboard.filtering(), "%s applies the filter too", k)
	}
	none := keys(t, typeText(t, keys(t, m, "/"), "nomatch"), "enter")
	require.Empty(t, none.dashboard.filter.Value(), "a filter without matches is dropped")
	require.Len(t, visibleNames(none), 3)
	require.Equal(t, "gamma", keys(t, filterTo(t, m, "gam"), "esc").dashboard.selected().Name, "clearing keeps the selection")

	pasted := send(t, keys(t, m, "/"), tea.PasteMsg{Content: "gam"})
	require.Equal(t, []string{"gamma"}, visibleNames(pasted), "a paste filters like typing")
}

func TestActionsUseSelectedProject(t *testing.T) {
	c := stubCommands(t)
	projects := sampleProjects()
	projects[2].Detail = &ProjectDetail{XHProfMode: types.XHProfModeXHGui}
	m := filterTo(t, newModel(t, 120, 24, projects...), "gamma")
	require.Equal(t, "gamma", m.dashboard.selected().Name)
	keys(t, m, "s")
	keys(t, m, "S")
	keys(t, m, "R")
	require.Equal(t, []string{"start gamma", "stop gamma", "restart gamma"}, c.execs)

	c.execs = nil
	keys(t, m, "l")
	keys(t, m, "m")
	keys(t, m, "x")
	keys(t, m, "C")
	require.Equal(t, []string{"/home/user/work/gamma launch", "/home/user/work/gamma mailpit", "/home/user/work/gamma xhgui", "config"}, c.execs)

	c.execs = nil
	keys(t, filterTo(t, newModel(t, 120, 24, sampleProjects()...), "gamma"), "x")
	require.Empty(t, c.execs, "ddev xhgui would start the project, then fail outside xhgui mode")
}

// TestOperationKeepsView checks that the view a command ran from stays
// behind it, since the command has the terminal while it runs.
func TestOperationKeepsView(t *testing.T) {
	c := stubCommands(t)
	m := keys(t, newModel(t, 120, 24, sampleProjects()...), "?", "s")
	require.Equal(t, viewDashboard, m.view)
	require.False(t, m.help.ShowAll, "the help closes, so the screen the command returns to is plain")
	require.Equal(t, "Starting alpha", c.ops[0].title)
}

func TestOpenKeys(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	c := stubCommands(t)
	want := []string{
		"directory|/home/user/apps/alpha",
		"VS Code|" + editorURL("vscode", "/home/user/apps/alpha"),
		"PhpStorm|/home/user/apps/alpha",
	}
	m := newModel(t, 120, 24, sampleProjects()...)
	keys(t, m, "o")
	keys(t, m, "v")
	keys(t, m, "p")
	require.Equal(t, want, c.opens)

	c.opens = nil
	d := detailModel(t, 120, 24, sampleDetail())
	keys(t, d, "o")
	keys(t, d, "v")
	keys(t, d, "p")
	require.Equal(t, want, c.opens)
}

func TestOpenedMsg(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	require.Contains(t, screen(send(t, m, openedMsg{what: "VS Code"})), "Opened VS Code")
	require.Contains(t, screen(send(t, m, openedMsg{what: "directory", err: errors.New("boom")})), "Open directory failed: boom")
}

func TestRefreshKeepsSelectionAndFilter(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	m = keys(t, m, "j")
	require.Equal(t, "beta", m.dashboard.selected().Name)

	// A new project sorts before beta; beta stays selected.
	more := append([]ProjectInfo{{Name: "aaa", Status: ddevapp.SiteStopped, AppRoot: "/home/user/aaa"}}, sampleProjects()...)
	m = send(t, m, projectsLoadedMsg{projects: more})
	require.Equal(t, "beta", m.dashboard.selected().Name)

	// When beta goes away, the cursor stays in range.
	m = send(t, m, projectsLoadedMsg{projects: more[:2]})
	require.NotNil(t, m.dashboard.selected())

	for _, applied := range []bool{false, true} {
		f := typeText(t, keys(t, newModel(t, 120, 24, sampleProjects()...), "/"), "a")
		if applied {
			// While typing a filter, j is part of it, so move only after.
			f = keys(t, f, "enter", "j")
		}
		selected := f.dashboard.selected().Name
		before := visibleNames(f)

		f = send(t, f, projectsLoadedMsg{projects: sampleProjects()})
		require.Equal(t, before, visibleNames(f), "applied %v: the refresh shows the same matches at once", applied)
		require.Equal(t, selected, f.dashboard.selected().Name, "applied %v", applied)
		require.Equal(t, !applied, f.dashboard.filtering())
		require.Equal(t, "a", f.dashboard.filter.Value())
	}
}

// listedProjects returns sampleProjects as read from their configuration,
// before their status is known.
func listedProjects() []ProjectInfo {
	projects := sampleProjects()
	for i := range projects {
		projects[i].Status, projects[i].URL = statusLoading, ""
		projects[i].Detail = &ProjectDetail{Name: projects[i].Name, Status: statusLoading, Type: projects[i].Type, AppRoot: projects[i].AppRoot, PHPVersion: "8.4", DatabaseType: "mariadb", FromConfig: true}
	}
	return projects
}

func TestProjectsListedBeforeStatus(t *testing.T) {
	m := send(t, newModel(t, 120, 24), projectsListedMsg{projects: listedProjects()})
	view := screen(m)
	require.NotContains(t, view, "Loading projects...")
	require.Contains(t, view, "gamma")
	require.Contains(t, view, "laravel")
	require.NotContains(t, view, ddevapp.SiteRunning)
	require.Contains(t, view, "gamma  ◌ loading")
	require.Contains(t, view, "SSH agent ◌ loading · Router ◌ loading")
	require.True(t, m.busy(), "the spinner runs until the status is known")

	m = keys(t, m, "j")
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.Contains(t, screen(m), ddevapp.SiteRunning)
	require.False(t, m.busy())
	require.Equal(t, "beta", m.dashboard.selected().Name, "the selection survives the status arriving")

	// A failure to load the status leaves the listed projects up, without
	// the spinner, loading again with the next refresh.
	failed := send(t, newModel(t, 120, 24), projectsListedMsg{projects: listedProjects()}, projectsLoadedMsg{err: errors.New("Docker error")})
	require.Contains(t, screen(failed), "gamma  ? unknown")
	require.Contains(t, screen(failed), "Router ? unknown")
	require.Contains(t, screen(failed), "Error loading projects: Docker error")
	require.False(t, failed.busy())

	noProjects := send(t, newModel(t, 120, 24), projectsLoadedMsg{err: errors.New("Docker error")})
	require.Contains(t, screen(noProjects), "Error: Docker error")
	require.False(t, noProjects.busy(), "an error shows without the spinner")
}

func TestCurrentProject(t *testing.T) {
	m := NewAppModel()
	m.dashboard.cwd = "/home/user/work/gamma/web/sites"
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 24}, projectsListedMsg{projects: listedProjects()})
	require.Equal(t, viewDashboard, m.view, "started inside a project, the TUI opens on the list")
	require.Equal(t, "gamma", m.dashboard.selected().Name, "with that project selected")
	require.Contains(t, screen(m), "· * current directory")
	require.Regexp(t, `(?m)^> gamma\* +◌ loading`, screen(m), "the mark keeps the columns aligned")
	require.Contains(t, screen(keys(t, m, "g")), "  gamma*", "the mark stays when the selection moves")

	// The status that follows keeps it selected, but later refreshes keep
	// the user's own selection.
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, "gamma", m.dashboard.selected().Name)
	m = send(t, keys(t, m, "g"), projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, "alpha", m.dashboard.selected().Name)

	// Outside any project, the first project is selected.
	outside := newModel(t, 120, 24, sampleProjects()...)
	require.Equal(t, "alpha", outside.dashboard.selected().Name)
	require.NotContains(t, screen(outside), "current directory")

	// The mark follows a name shorter than the column.
	beta := NewAppModel()
	beta.dashboard.cwd = "/home/user/apps/beta"
	beta = send(t, beta, tea.WindowSizeMsg{Width: 120, Height: 24}, projectsListedMsg{projects: listedProjects()})
	require.Regexp(t, `(?m)^> beta\*  +◌ loading`, screen(beta))
}

func TestConfigSelectsNewProject(t *testing.T) {
	var withoutGamma []ProjectInfo
	for _, p := range sampleProjects() {
		if p.Name != "gamma" {
			withoutGamma = append(withoutGamma, p)
		}
	}
	m := NewAppModel()
	m.dashboard.cwd = "/home/user/work/gamma"
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 24}, projectsLoadedMsg{projects: withoutGamma})
	require.Equal(t, "alpha", m.dashboard.selected().Name)

	c := stubCommands(t)
	keys(t, m, "C")
	require.Equal(t, []string{"config"}, c.execs, "C runs ddev config in the current directory")
	require.Equal(t, execFinishedMsg{title: "ddev config", done: "Configured the current directory", selectCurrent: true}, c.ops[0].finished(nil))

	failed := send(t, m, execFinishedMsg{title: "ddev config", err: errors.New("exit status 1")}, projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, "alpha", failed.dashboard.selected().Name, "a failed config leaves the selection")

	m = send(t, m, execFinishedMsg{title: "ddev config", selectCurrent: true}, projectsLoadedMsg{projects: withoutGamma})
	require.Equal(t, "alpha", m.dashboard.selected().Name, "a load from before the config does not list it yet")
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, "gamma", m.dashboard.selected().Name, "the configured project is selected once listed")
	m = send(t, keys(t, m, "g"), projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, "alpha", m.dashboard.selected().Name, "and later refreshes keep the user's selection")
}

func TestDetailBeforeStatus(t *testing.T) {
	m := NewAppModel()
	m.dashboard.cwd = "/home/user/work/gamma"
	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 24}, projectsListedMsg{projects: listedProjects()})
	m = keys(t, m, "enter")
	require.Equal(t, viewDetail, m.view)
	require.True(t, m.detail.loading)
	view := screen(m)
	require.NotContains(t, view, "Loading project details...", "the detail opens on what the configuration says")
	require.Contains(t, view, "gamma  ◌ loading")
	require.Contains(t, view, row("PHP", "8.4"))
	require.Contains(t, view, row("Xdebug", "…"))
	require.Contains(t, view, row("URLs", "loading"))
	require.Contains(t, view, "mariadb")
	require.Equal(t, 2, strings.Count(view, " Loading..."), "services and add-ons wait for the full detail")

	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.Equal(t, viewDetail, m.view)
	gamma := sampleDetail()
	gamma.Name, gamma.AppRoot = "gamma", "/home/user/work/gamma"
	m = send(t, m, projectDetailLoadedMsg{appRoot: gamma.AppRoot, detail: gamma})
	require.Contains(t, screen(m), "Services")
}

func TestFormatProjectPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	slashed := filepath.ToSlash(home)
	require.Equal(t, "~", formatProjectPath(home))
	require.Equal(t, "~/apps/alpha", formatProjectPath(filepath.Join(home, "apps", "alpha")))
	require.Equal(t, slashed+"2/alpha", formatProjectPath(filepath.Join(home+"2", "alpha")), "only home itself is shortened, not a sibling")
	require.Empty(t, formatProjectPath(""))
}

func TestCurrentProjectRoot(t *testing.T) {
	projects := []ProjectInfo{{AppRoot: "/srv/site"}, {AppRoot: "/srv/site/nested"}, {AppRoot: "/srv/site2"}}
	require.Equal(t, "/srv/site", currentProjectRoot(projects, "/srv/site"))
	require.Equal(t, "/srv/site", currentProjectRoot(projects, "/srv/site/web"))
	require.Equal(t, "/srv/site/nested", currentProjectRoot(projects, "/srv/site/nested/x"), "the innermost project wins")
	require.Equal(t, "/srv/site2", currentProjectRoot(projects, "/srv/site2"))
	require.Empty(t, currentProjectRoot(projects, "/srv"))
	require.Empty(t, currentProjectRoot(projects, "/srv/site-other"))
}

func TestConfirmations(t *testing.T) {
	c := stubCommands(t)
	m := newModel(t, 120, 24, sampleProjects()...)
	for k, want := range map[string]string{"a": "start --all", "A": "stop --all", "P": "poweroff"} {
		asked := keys(t, m, k)
		require.NotNil(t, asked.confirm, k)
		require.Contains(t, screen(asked), "y confirm")
		require.NotContains(t, screen(asked), "delete without snapshot")

		c.execs = nil
		cancelled := keys(t, asked, "n")
		require.Nil(t, cancelled.confirm)
		require.Empty(t, c.execs)
		require.Equal(t, viewDashboard, cancelled.view)

		keys(t, asked, "O")
		require.Empty(t, c.execs, "O only applies to a delete")

		confirmed := keys(t, asked, "y")
		require.Equal(t, []string{want}, c.execs)
		require.Nil(t, confirmed.confirm)
	}
	require.Contains(t, screen(keys(t, m, "a")), "Start all 3 projects?")
	require.Nil(t, keys(t, newModel(t, 120, 24, []ProjectInfo{}...), "a").confirm, "nothing to start")
}

func TestDelete(t *testing.T) {
	origOmit := globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete
	t.Cleanup(func() { globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete = origOmit })
	c := stubCommands(t)

	for name, m := range map[string]AppModel{
		"dashboard": newModel(t, 120, 24, sampleProjects()...),
		"detail":    detailModel(t, 120, 24, sampleDetail()),
	} {
		for _, omit := range []bool{false, true} {
			globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete = omit
			asked := keys(t, m, "D")
			view := screen(asked)
			require.Contains(t, view, "Delete project 'alpha' and its database?", name)
			require.Equal(t, omit, strings.Contains(view, "No snapshot will be made (omit_snapshot_on_delete)"), name)
			require.Equal(t, !omit, strings.Contains(view, "A database snapshot will be made first."), name)
		}
		globalconfig.DdevGlobalConfig.OmitSnapshotOnDelete = false
		asked := keys(t, m, "D")
		for k, want := range map[string]string{"y": "delete -y alpha", "O": "delete -y --omit-snapshot alpha"} {
			c.execs = nil
			done := keys(t, asked, k)
			require.Equal(t, []string{want}, c.execs, "%s in %s", k, name)
			require.Equal(t, viewDashboard, done.view, "a deleted project has no detail view to return to")
			require.Empty(t, done.detail.root, "nor a detail for a load still running to update")
		}
		// o opens the project directory elsewhere, so a habitual press cancels.
		for _, k := range []string{"n", "o", "esc"} {
			c.execs, c.opens = nil, nil
			cancelled := keys(t, asked, k)
			require.Empty(t, c.execs)
			require.Empty(t, c.opens)
			require.Nil(t, cancelled.confirm)
			require.Equal(t, m.view, cancelled.view)
			require.NotContains(t, screen(cancelled), "Delete project")
		}
	}
	require.Nil(t, keys(t, newModel(t, 120, 24, []ProjectInfo{}...), "D").confirm, "nothing to delete")
}

func TestStatusMessages(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	m, cmd := m.setStatus("first")
	require.NotNil(t, cmd)
	first := m.statusID
	m, _ = m.setStatus("second")
	m = send(t, m, statusExpiredMsg{id: first})
	require.Contains(t, screen(m), "second", "an older expiry leaves a newer message")
	m = send(t, m, statusExpiredMsg{id: m.statusID})
	require.NotContains(t, screen(m), "second")

	m = m.setError("broken")
	m = send(t, m, statusExpiredMsg{id: first}, statusExpiredMsg{id: m.statusID - 1})
	require.Contains(t, screen(m), "broken", "errors stay until replaced")
	require.NotContains(t, screen(keys(t, m, "j")), "broken", "or until a key is pressed")

	warned := send(t, m, projectsLoadedMsg{projects: sampleProjects(), warning: "Something went wrong with /x"})
	require.Equal(t, "broken", warned.status, "what a load logged does not replace an error")
	warned = send(t, warned.setError(""), projectsLoadedMsg{projects: sampleProjects(), warning: "Something went wrong with /x"})
	require.Empty(t, warned.status, "nor shows again on the next refresh")
	warned = send(t, warned, projectsLoadedMsg{projects: sampleProjects()}, projectsLoadedMsg{projects: sampleProjects(), warning: "Something went wrong with /x"})
	require.Equal(t, "Something went wrong with /x", warned.status, "until a load without it")

	long := m.setError(strings.Repeat("a long error ", 50))
	require.Equal(t, 4, lipgloss.Height(long.footer()), "a long message keeps to two lines between the divider and the help")
	require.Contains(t, screen(long), "…")
	require.Contains(t, screen(long), "q quit")
}

func TestRefresh(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	m, cmd := sendCmd(t, m, press("r"))
	require.NotNil(t, cmd)
	require.True(t, m.refreshing)
	require.True(t, m.dashboard.loading)
	require.Contains(t, screen(m), m.spinner.View())

	// The periodic refresh waits for a load in flight instead of piling up.
	_, cmd = sendCmd(t, m, tickMsg{})
	require.NotNil(t, cmd, "the ticker keeps going")
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.False(t, m.refreshing)
	require.False(t, m.dashboard.loading)
	m = send(t, m, tickMsg{})
	require.True(t, m.dashboard.loading)
	require.False(t, m.refreshing, "the periodic refresh shows no spinner")
}

func TestExecFinished(t *testing.T) {
	m := newModel(t, 120, 24, sampleProjects()...)
	m = send(t, m, execFinishedMsg{title: "Starting alpha"})
	require.True(t, m.dashboard.loading, "the command may have changed the projects")
	require.Contains(t, screen(m), "Starting alpha: done", "a command that worked says so")
	require.Contains(t, screen(send(t, m, execFinishedMsg{title: "SSH into alpha", done: "SSH session ended"})), "SSH session ended")

	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()}, execFinishedMsg{title: "Starting alpha", err: errors.New("exit status 1")})
	require.Contains(t, screen(m), "Starting alpha failed: exit status 1")

	// A load that started before the command ended may miss what it changed,
	// so another one follows it.
	m = send(t, newModel(t, 120, 24, sampleProjects()...), tickMsg{}, execFinishedMsg{title: "ddev config"})
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.True(t, m.dashboard.loading)
	m = send(t, m, projectsLoadedMsg{projects: sampleProjects()})
	require.False(t, m.dashboard.loading)
}
