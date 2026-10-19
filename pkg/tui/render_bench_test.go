// The benchmarks in this file measure what `ddev tui` costs to redraw: the
// list of 10 to 500 projects, the detail and the log stream. They check
// nothing, and neither `go test` nor CI runs them, so run them on main and on
// a change to rendering, then compare:
//
//	go test -run '^$' -bench . ./pkg/tui
package tui

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// benchFrames measures the spinner redrawing m: one Update and one View.
func benchFrames(b *testing.B, m AppModel) {
	b.Helper()
	m.refreshing = true
	tick := m.spinner.Tick().(spinner.TickMsg)
	b.ReportAllocs()
	for b.Loop() {
		m = update(m, tick)
		_ = m.View()
	}
}

// update is send without its checks, which would be measured too.
func update(m AppModel, msg tea.Msg) AppModel {
	updated, _ := m.Update(msg)
	return updated.(AppModel)
}

func BenchmarkDashboardFrame(b *testing.B) {
	for _, n := range []int{10, 100, 500} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			benchFrames(b, newModel(b, 160, 50, manyProjects(n)...))
		})
	}
}

// BenchmarkDashboardRefresh measures the periodic refresh, which replaces
// every project.
func BenchmarkDashboardRefresh(b *testing.B) {
	for _, n := range []int{10, 100, 500} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			projects := manyProjects(n)
			m := newModel(b, 160, 50, projects...)
			msg := projectsLoadedMsg{projects: projects, router: m.router}
			b.ReportAllocs()
			for b.Loop() {
				m = update(m, msg)
				_ = m.View()
			}
		})
	}
}

func BenchmarkDetailFrame(b *testing.B) {
	benchFrames(b, detailModel(b, 160, 50, busyDetail()))
}

// longLog returns the log view with n lines of output.
func longLog(b *testing.B, n int) AppModel {
	b.Helper()
	m, _ := newModel(b, 160, 50, manyProjects(3)...).apply(logsAction{name: "logs"})
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("%d %s", i, longText)
	}
	return send(b, m, streamLinesMsg{id: m.stream.id, lines: lines})
}

func BenchmarkStreamFrame(b *testing.B) {
	benchFrames(b, longLog(b, 2000))
}

// BenchmarkStreamLine measures a line of output arriving in a long log.
func BenchmarkStreamLine(b *testing.B) {
	m := longLog(b, 4000)
	msg := streamLinesMsg{id: m.stream.id, lines: []string{longText}}
	b.ReportAllocs()
	for b.Loop() {
		m = update(m, msg)
		_ = m.View()
	}
}
