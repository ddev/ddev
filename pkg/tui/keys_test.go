package tui

import (
	"reflect"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

// helpStates returns a model for each view in which the most keys apply.
func helpStates(t *testing.T) map[string]AppModel {
	t.Helper()
	dashboard := newModel(t, 200, 60, sampleProjects()...)
	stream, _ := dashboard.apply(logsAction{name: "alpha"})
	detail := sampleDetail()
	detail.XHProfMode = types.XHProfModeXHGui
	return map[string]AppModel{
		"dashboard": dashboard,
		"filtered":  keys(t, typeText(t, keys(t, dashboard, "/"), "a"), "enter"),
		"filtering": typeText(t, keys(t, dashboard, "/"), "a"),
		"detail": send(t, detailModel(t, 200, 60, detail), xdebugStatusMsg{appRoot: detail.AppRoot},
			latestAddonsMsg{appRoot: detail.AppRoot, versions: map[string]string{"ddev/ddev-redis": "v2.1.0"}}),
		"stream":  stream,
		"confirm": keys(t, dashboard, "D"),
	}
}

func enabledBindings(groups [][]key.Binding) []key.Binding {
	var out []key.Binding
	for _, group := range groups {
		for _, b := range group {
			if b.Enabled() {
				out = append(out, b)
			}
		}
	}
	return out
}

func TestHelpListsEveryKey(t *testing.T) {
	shown := map[string]bool{}
	for _, m := range helpStates(t) {
		for _, b := range enabledBindings(m.helpKeys().FullHelp()) {
			shown[b.Help().Key] = true
		}
	}
	v := reflect.ValueOf(DefaultKeyMap())
	for i := range v.NumField() {
		b := v.Field(i).Interface().(key.Binding)
		require.True(t, shown[b.Help().Key], "%s (%s) is in no view's help", v.Type().Field(i).Name, b.Help().Key)
	}
}

// TestNoConflictingKeys checks that within a view, no two bindings that can
// act share a key, counting the viewport bindings that the help leaves out.
func TestNoConflictingKeys(t *testing.T) {
	for name, m := range helpStates(t) {
		bindings := enabledBindings(m.helpKeys().FullHelp())
		switch {
		case m.confirm != nil:
		case m.view == viewDetail:
			vp := m.detail.vp.KeyMap
			bindings = append(bindings, vp.Up, vp.Down, vp.PageUp, vp.PageDown, vp.HalfPageUp, vp.HalfPageDown, vp.Left, vp.Right)
		case m.view == viewStream:
			vp := m.stream.vp.KeyMap
			bindings = append(bindings, vp.Up, vp.Down, vp.PageUp, vp.PageDown, vp.HalfPageUp, vp.HalfPageDown, vp.Left, vp.Right)
		}
		owner := map[string]string{}
		for _, b := range bindings {
			if !b.Enabled() {
				continue
			}
			desc := b.Help().Desc
			for _, k := range b.Keys() {
				if prev, ok := owner[k]; ok && prev != desc {
					require.Failf(t, "key conflict", "%s: %q is bound to both %q and %q", name, k, prev, desc)
				}
				owner[k] = desc
			}
		}
	}
}

func TestQuit(t *testing.T) {
	m := newModel(t, 80, 24, sampleProjects()...)
	_, cmd := sendCmd(t, m, press("q"))
	require.IsType(t, tea.QuitMsg{}, cmd())

	// While filtering, q is typed into the filter, but ctrl+c still quits.
	m = typeText(t, keys(t, m, "/"), "q")
	require.Equal(t, "q", m.dashboard.filter.Value())
	_, cmd = sendCmd(t, m, press("ctrl+c"))
	require.IsType(t, tea.QuitMsg{}, cmd())

	// ctrl+c also quits from a confirmation, rather than cancelling it.
	_, cmd = sendCmd(t, keys(t, newModel(t, 80, 24, sampleProjects()...), "P"), press("ctrl+c"))
	require.IsType(t, tea.QuitMsg{}, cmd())
}

func TestHelpHidesKeysThatCannotAct(t *testing.T) {
	require.NotContains(t, screen(newModel(t, 200, 40)), "s start", "no project is selected")

	d := detailModel(t, 200, 40, ProjectDetail{Name: "beta", Status: ddevapp.SiteStopped, AppRoot: "/home/user/apps/beta"})
	help := screen(keys(t, d, "?"))
	require.NotContains(t, help, "ssh")
	require.NotContains(t, help, "toggle xdebug")
	require.Contains(t, help, "start")
}

func TestHelpToggle(t *testing.T) {
	m := newModel(t, 200, 40, sampleProjects()...)
	require.Contains(t, screen(m), "? more")
	require.NotContains(t, screen(m), "open in PhpStorm")

	m = keys(t, m, "?")
	require.True(t, m.help.ShowAll)
	require.Contains(t, screen(m), "open in PhpStorm")
	require.Contains(t, screen(m), "? close help")
	require.Contains(t, screen(m), "alpha", "the full help shares the screen with the list")

	require.False(t, keys(t, m, "?").help.ShowAll)
	require.False(t, keys(t, m, "esc").help.ShowAll)
	require.Equal(t, viewDashboard, keys(t, m, "esc").view)
}

func TestShortHelpFitsOneLine(t *testing.T) {
	for name, m := range helpStates(t) {
		for _, w := range []int{30, 80} {
			m = send(t, m, tea.WindowSizeMsg{Width: w, Height: 30})
			if m.help.ShowAll {
				continue
			}
			short := m.helpView()
			require.NotContains(t, short, "\n", name)
			require.LessOrEqual(t, ansi.StringWidth(short), w, name)
		}
	}
}
