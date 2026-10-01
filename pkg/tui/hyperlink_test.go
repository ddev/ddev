package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/stretchr/testify/require"
)

// HasTermHyperlinks caches its result on first use, so the override must be
// in place before any test runs.
func init() {
	_ = os.Setenv("FORCE_HYPERLINK", "1")
}

const (
	osc8Open  = "\x1b]8;;"
	osc8Close = "\x1b]8;;\x1b\\"
)

// requireBalancedLinks fails if s has an unclosed OSC 8 hyperlink.
func requireBalancedLinks(t *testing.T, s string) {
	t.Helper()
	opens := strings.Count(s, osc8Open) - strings.Count(s, osc8Close)
	closes := strings.Count(s, osc8Close)
	require.Equal(t, opens, closes, "every hyperlink needs a terminator: %q", s)
}

func TestDashboardHyperlinks(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.projects = []ProjectInfo{{
		Name: "site-a", Status: ddevapp.SiteRunning, Type: "drupal",
		URL: "https://site-a.ddev.site", AppRoot: "/home/user/site-a",
	}}

	out := m.buildDashboardContent()
	require.Contains(t, out, "\x1b]8;;file:///home/user/site-a\x1b\\")
	require.Contains(t, out, "\x1b]8;;https://site-a.ddev.site\x1b\\")
	requireBalancedLinks(t, out)
}

func TestDashboardHyperlinksTruncated(t *testing.T) {
	m := NewAppModel()
	m.width = 70
	m.projects = []ProjectInfo{{
		Name: "site-a", Status: ddevapp.SiteRunning, Type: "drupal",
		URL:     "https://site-a-with-a-very-long-hostname.ddev.site",
		AppRoot: "/home/user/some/very/deeply/nested/project/directory/site-a",
	}}

	out := m.buildDashboardContent()
	require.Contains(t, out, "file:///home/user/some/very/deeply/nested/project/directory/site-a",
		"full path stays in the link target even when the text is truncated")
	requireBalancedLinks(t, out)
}

func TestDetailHyperlinks(t *testing.T) {
	m := NewAppModel()
	m.width = 100
	m.detail = &ProjectDetail{
		AppRoot:    "/home/user/site-a",
		URLs:       []string{"https://site-a.ddev.site"},
		MailpitURL: "https://site-a.ddev.site:8026",
	}

	out := m.buildDetailContent()
	require.Contains(t, out, "Approot:")
	require.Contains(t, out, "\x1b]8;;file:///home/user/site-a\x1b\\")
	require.Contains(t, out, "\x1b]8;;https://site-a.ddev.site\x1b\\")
	require.Contains(t, out, "\x1b]8;;https://site-a.ddev.site:8026\x1b\\")
	requireBalancedLinks(t, out)
}

func TestEditorURL(t *testing.T) {
	require.Equal(t, "vscode://file/home/user/my%20site", editorURL("vscode", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=/home/user/my%20site", editorURL("phpstorm", "/home/user/my site"))
	require.Equal(t, "vscode://file/C:/Users/me/site", editorURL("vscode", "C:/Users/me/site"))
	require.Empty(t, editorURL("unknown", "/home/user"))
}

func TestDetailEditorLinks(t *testing.T) {
	m := NewAppModel()
	m.width = 100
	m.detail = &ProjectDetail{AppRoot: "/home/user/site-a"}

	out := m.buildDetailContent()
	require.Contains(t, out, "\x1b]8;;vscode://file/home/user/site-a\x1b\\")
	require.Contains(t, out, "\x1b]8;;phpstorm://open?file=/home/user/site-a\x1b\\")
}

func TestOpenKeys(t *testing.T) {
	for _, k := range []rune{'o', 'v', 'p'} {
		d := NewAppModel()
		d.loading = false
		d.projects = []ProjectInfo{{Name: "mysite", Status: ddevapp.SiteRunning, AppRoot: "/tmp/mysite"}}
		_, cmd := d.Update(tea.KeyPressMsg{Code: k, Text: string(k)})
		require.NotNil(t, cmd, "%c on the dashboard should return an open command", k)

		m := NewAppModel()
		m.viewMode = viewDetail
		detail := sampleDetail()
		m.detail = &detail
		_, cmd = m.Update(tea.KeyPressMsg{Code: k, Text: string(k)})
		require.NotNil(t, cmd, "%c in the detail view should return an open command", k)
	}
}

func TestOpenedMsgStatus(t *testing.T) {
	m := NewAppModel()
	updated, _ := m.Update(openedMsg{what: "VS Code"})
	require.Equal(t, "Opened VS Code", updated.(AppModel).statusMsg)

	updated, _ = m.Update(openedMsg{what: "directory", err: fmt.Errorf("boom")})
	require.Contains(t, updated.(AppModel).statusMsg, "failed: boom")
}
