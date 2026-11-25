package webui

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/stretchr/testify/require"
)

// TestOpenURLOnlyDesktop checks the link route of `ddev gui --desktop`,
// which the GUI app's window uses because a webview opens no new
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

// TestStopWithStdin checks that a host command still runs while the server
// reads the GUI app's pipe on stdin, and that closing the pipe stops the
// server. Docker is not needed, so CI on Windows catches the hang.
func TestStopWithStdin(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdin := os.Stdin
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = w.Close()
	})
	os.Stdin = r

	s := newTestServer(t)
	served := make(chan error, 1)
	go func() { served <- s.Serve() }()
	s.StopWithStdin()
	// The hang needs the server's read to be pending when the child starts.
	time.Sleep(500 * time.Millisecond)

	// Without the fix, Git for Windows hangs here, but a Go child does not.
	ran := make(chan error, 1)
	go func() {
		_, err := exec.RunHostCommand("git", "--version")
		ran <- err
	}()
	select {
	case err := <-ran:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("a host command hung while the server read stdin")
	}

	require.NoError(t, w.Close())
	select {
	case err := <-served:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not stop when stdin closed")
	}
}
