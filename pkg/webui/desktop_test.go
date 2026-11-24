package webui

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenURLOnlyDesktop checks the link route of `ddev serve --desktop`,
// which the desktop app's window uses because a webview opens no new
// windows.
func TestOpenURLOnlyDesktop(t *testing.T) {
	s := newTestServer(t)
	rec := request(s, "POST", "/api/open-url", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token)
	})
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code, "a browser opens links itself, so no POST route exists")

	d := newTestServer(t)
	d.SetDesktop()
	for _, u := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "vscode://file/tmp", "https://"} {
		rec = postForm(d, "/api/open-url", url.Values{"url": {u}})
		require.Equal(t, http.StatusBadRequest, rec.Code, u)
	}
}

// TestIsUp checks which projects the tray and the page offer Stop for.
func TestIsUp(t *testing.T) {
	for status, want := range map[string]bool{
		"running": true, "unhealthy": true, "starting": true,
		"stopped": false, "paused": false, "project directory missing": false, "": false,
	} {
		require.Equal(t, want, IsUp(status), status)
	}
}
