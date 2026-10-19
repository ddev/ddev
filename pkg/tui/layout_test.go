package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

var longText = strings.Repeat("very-long-segment-", 12)

// manyProjects returns n projects with long names, paths and URLs.
func manyProjects(n int) []ProjectInfo {
	projects := make([]ProjectInfo, n)
	for i := range projects {
		name := fmt.Sprintf("project-%02d-%s", i, longText[:i%40])
		projects[i] = ProjectInfo{
			Name: name, Status: ddevapp.SiteRunning, Type: "drupal11",
			URL:     "https://" + name + ".ddev.site",
			AppRoot: "/home/user/" + longText + name,
		}
		if i%3 == 1 {
			projects[i].Status, projects[i].URL = ddevapp.SiteStopped, ""
		}
	}
	return projects
}

// busyDetail returns a detail with long values, a tab and many services.
func busyDetail() ProjectDetail {
	d := sampleDetail()
	d.Docroot = "web\tpublic"
	d.URLs = append(d.URLs, "https://"+longText[:120]+".ddev.site")
	for i := range 15 {
		d.Services = append(d.Services, ServiceInfo{
			Name: fmt.Sprintf("svc%02d", i), Status: ddevapp.SiteRunning,
			URL: "https://" + longText + ".ddev.site:" + fmt.Sprint(8000+i), Info: longText,
		})
	}
	return d
}

// layoutStates builds each state the layout must fit, for a window size.
var layoutStates = map[string]func(t *testing.T, w, h int) AppModel{
	"loading": func(t *testing.T, w, h int) AppModel { return newModel(t, w, h) },
	"error": func(t *testing.T, w, h int) AppModel {
		return send(t, newModel(t, w, h), projectsLoadedMsg{err: errors.New(longText)})
	},
	"empty": func(t *testing.T, w, h int) AppModel {
		return send(t, newModel(t, w, h), projectsLoadedMsg{projects: []ProjectInfo{}})
	},
	"projects": func(t *testing.T, w, h int) AppModel { return newModel(t, w, h, manyProjects(40)...) },
	"help":     func(t *testing.T, w, h int) AppModel { return keys(t, newModel(t, w, h, manyProjects(40)...), "?") },
	"filtering": func(t *testing.T, w, h int) AppModel {
		return typeText(t, keys(t, newModel(t, w, h, manyProjects(40)...), "/"), "proj")
	},
	"filtered": func(t *testing.T, w, h int) AppModel {
		return keys(t, typeText(t, keys(t, newModel(t, w, h, manyProjects(40)...), "/"), "proj"), "enter")
	},
	"confirm": func(t *testing.T, w, h int) AppModel { return keys(t, newModel(t, w, h, manyProjects(40)...), "D") },
	"error status": func(t *testing.T, w, h int) AppModel {
		m := newModel(t, w, h, manyProjects(40)...).setError(longText)
		return send(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	},
	"detail loading": func(t *testing.T, w, h int) AppModel {
		m, _ := newModel(t, w, h, manyProjects(3)...).openDetail(manyProjects(1)[0])
		return send(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	},
	"detail":      func(t *testing.T, w, h int) AppModel { return detailModel(t, w, h, busyDetail()) },
	"detail help": func(t *testing.T, w, h int) AppModel { return keys(t, detailModel(t, w, h, busyDetail()), "?") },
	"stream": func(t *testing.T, w, h int) AppModel {
		m, _ := newModel(t, w, h, manyProjects(3)...).apply(logsAction{name: longText})
		for i := range 200 {
			m = send(t, m, streamLinesMsg{id: m.stream.id, lines: []string{fmt.Sprintf("%d %s", i, longText)}})
		}
		return m
	},
	"stream failed": func(t *testing.T, w, h int) AppModel {
		m, _ := newModel(t, w, h, manyProjects(3)...).apply(logsAction{name: "alpha"})
		return send(t, m, streamEndedMsg{id: m.stream.id, err: errors.New(longText)})
	},
}

func TestLayoutFitsWindow(t *testing.T) {
	for _, size := range [][2]int{{20, 6}, {40, 10}, {80, 24}, {120, 40}, {200, 60}} {
		for name, state := range layoutStates {
			m := state(t, size[0], size[1])
			requireFits(t, m, fmt.Sprintf("%s at %dx%d", name, size[0], size[1]))
		}
	}
}

func TestLayoutFollowsResize(t *testing.T) {
	for name, state := range layoutStates {
		m := state(t, 120, 40)
		for _, size := range [][2]int{{60, 15}, {30, 8}, {150, 50}, {80, 24}} {
			m = send(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			requireFits(t, m, fmt.Sprintf("%s resized to %dx%d", name, size[0], size[1]))
		}
	}
}

// Update skips the layout for a spinner frame, so a frame must not change it.
func TestSpinnerFrameKeepsLayout(t *testing.T) {
	for name, state := range layoutStates {
		m := state(t, 80, 24)
		m.refreshing = true
		m.layout()
		m = send(t, m, m.spinner.Tick())
		laid := m
		laid.layout()
		require.Equal(t, laid.View().Content, m.View().Content, name)
	}
}

func TestResizeKeepsSelectionVisible(t *testing.T) {
	m := keys(t, newModel(t, 100, 30, manyProjects(40)...), "G")
	items := m.dashboard.items
	last := items[len(items)-1].Name
	require.Equal(t, last, m.dashboard.selected().Name)
	for _, h := range []int{10, 6, 45, 24} {
		m = send(t, m, tea.WindowSizeMsg{Width: 100, Height: h})
		require.Equal(t, last, m.dashboard.selected().Name)
		require.Contains(t, screen(m), "> "+last[:19], "height %d", h)
	}
}

func TestDetailFollowsWidth(t *testing.T) {
	m := detailModel(t, 200, 40, busyDetail())
	longURL := "https://" + longText[:120] + ".ddev.site"
	require.Contains(t, screen(m), longURL)

	m = send(t, m, tea.WindowSizeMsg{Width: 60, Height: 40})
	require.NotContains(t, screen(m), longURL)
	require.Contains(t, screen(m), longURL[:40], "the URL is cut to the new width, not dropped")

	m = send(t, m, tea.WindowSizeMsg{Width: 200, Height: 40})
	require.Contains(t, screen(m), longURL, "growing the window shows the whole URL again")
}

func TestFullHelpFallsBackWhenTooTall(t *testing.T) {
	m := keys(t, newModel(t, 120, 12, manyProjects(5)...), "?")
	require.True(t, m.help.ShowAll)
	require.Contains(t, screen(m), "? close help", "the short help says how to close it")
	require.NotContains(t, screen(m), "open in PhpStorm", "the full help does not fit in 12 lines")

	m = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Contains(t, screen(m), "open in PhpStorm")
}
