package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func embeddedRequest(s *Server, method, target, body string, mod func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Host = "wails.localhost"
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func withDashboardHeader(r *http.Request) { r.Header.Set(dashboardHeader, "1") }

// TestEmbeddedSecurity checks the webview mode, which has no token but
// still refuses a state change another page could send.
func TestEmbeddedSecurity(t *testing.T) {
	s := NewEmbedded("ddev")

	rec := embeddedRequest(s, "GET", "/", "", nil)
	require.Equal(t, http.StatusOK, rec.Code, "the webview needs no token")
	require.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")

	rec = embeddedRequest(s, "POST", "/api/actions/nothing", "", nil)
	require.Equal(t, http.StatusForbidden, rec.Code, "a POST without the dashboard header")

	rec = embeddedRequest(s, "POST", "/api/actions/nothing", "", withDashboardHeader)
	require.Equal(t, http.StatusNotFound, rec.Code, "the router answers once the check passes")

	// A cross-origin page would need a preflight, which must not succeed.
	rec = embeddedRequest(s, "OPTIONS", "/api/actions/poweroff", "", nil)
	require.GreaterOrEqual(t, rec.Code, 400)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestOpenURLOnlyEmbedded(t *testing.T) {
	s := newTestServer(t)
	rec := request(s, "POST", "/api/open-url", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token)
	})
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code, "a browser opens links itself, so no POST route exists")

	e := NewEmbedded("ddev")
	for _, u := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "vscode://file/tmp", "https://"} {
		rec = embeddedRequest(e, "POST", "/api/open-url", "url="+u, withDashboardHeader)
		require.Equal(t, http.StatusBadRequest, rec.Code, u)
	}
}

// TestEmbeddedPostRunsJob starts a job the way a tray menu does and checks
// that OnJobDone reports it.
func TestEmbeddedPostRunsJob(t *testing.T) {
	for _, tc := range []struct{ exit, wantErr string }{{"0", ""}, {"1", "exit status 1"}} {
		s := NewEmbedded("")
		fakeDdev(t, s, tc.exit)
		done := make(chan [3]string, 1)
		s.OnJobDone = func(title, project, errText string) { done <- [3]string{title, project, errText} }

		require.NoError(t, s.Post("/api/actions/stop-all"))
		select {
		case got := <-done:
			require.Equal(t, [3]string{"Stopping all projects", "", tc.wantErr}, got)
		case <-time.After(10 * time.Second):
			t.Fatal("OnJobDone was not called")
		}
		require.Error(t, s.Post("/api/actions/nothing"), "an unknown action")
	}
}
