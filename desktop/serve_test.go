package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChooseTarget(t *testing.T) {
	t.Setenv("DDEV_DESKTOP_MODE", "loopback")
	_, err := chooseTarget("")
	require.ErrorContains(t, err, "must be native or wsl")

	t.Setenv("DDEV_DESKTOP_MODE", "wsl")
	tgt, err := chooseTarget("")
	require.NoError(t, err)
	require.Equal(t, target{ddev: "ddev", wsl: true}, tgt)
	require.Equal(t, "ddev in WSL2", tgt.String())

	// The installer's shortcut names the distro it installed ddev in.
	t.Setenv("DDEV_DESKTOP_MODE", "")
	tgt, err = chooseTarget("Ubuntu-24.04")
	require.NoError(t, err)
	require.Equal(t, target{ddev: "ddev", wsl: true, distro: "Ubuntu-24.04"}, tgt)
	require.Equal(t, []string{"-d", "Ubuntu-24.04", "-e", "bash", "-lc", "command -v ddev"}, wslArgs(tgt.distro, "command -v ddev"))

	t.Setenv("DDEV_BINARY", "/opt/ddev/bin/ddev")
	tgt, err = chooseTarget("")
	require.NoError(t, err)
	require.Equal(t, target{ddev: "/opt/ddev/bin/ddev"}, tgt)
}

func TestArgValue(t *testing.T) {
	require.Equal(t, "Ubuntu", argValue([]string{"--wsl-distro=Ubuntu"}, "--wsl-distro"))
	require.Equal(t, "Debian", argValue([]string{"-psn_0_1", "--wsl-distro", "Debian"}, "--wsl-distro"))
	require.Empty(t, argValue([]string{"--wsl-distro"}, "--wsl-distro"))
	require.Empty(t, argValue(nil, "--wsl-distro"))
}

func TestReadStatusEvents(t *testing.T) {
	stream := "event: status\ndata: {\"projects\":[{\"name\":\"d11\",\"status\":\"running\",\"up\":true}],\"version\":\"v1.25.5\"}\n\n" +
		"event: other\ndata: {\"projects\":[]}\n\n" +
		"event: status\ndata: {\"error\":\"Docker is not reachable\"}\n\n"
	var got []*serveStatus
	err := readStatusEvents(strings.NewReader(stream), func(st *serveStatus) { got = append(got, st) })
	require.ErrorContains(t, err, "ended")
	require.Len(t, got, 2)
	require.Equal(t, []project{{Name: "d11", Status: "running", Up: true}}, got[0].Projects)
	require.Equal(t, "v1.25.5", got[0].Version)
	require.Equal(t, "Docker is not reachable", got[1].Error)
}

// TestJobNotifications checks which job ends the app notifies about.
func TestJobNotifications(t *testing.T) {
	c := &serveClient{changed: make(chan struct{}), jobs: map[int]bool{}}
	var ended []string
	c.OnJobDone = func(title, _, errText string) { ended = append(ended, title+"|"+errText) }

	changed := c.Changed()
	// Jobs that ended before the app started are not news.
	c.setStatus(&serveStatus{Jobs: []jobState{{ID: 1, Title: "old", Done: true}, {ID: 2, Title: "Starting d11"}}}, nil)
	require.Empty(t, ended)
	select {
	case <-changed:
	default:
		t.Fatal("Changed was not closed by a status")
	}
	require.Equal(t, []string{"Starting d11"}, c.RunningJobs())

	c.setStatus(&serveStatus{Jobs: []jobState{
		{ID: 1, Title: "old", Done: true},
		{ID: 2, Title: "Starting d11", Done: true, Err: "exit status 1"},
		// Started and ended between two events.
		{ID: 3, Title: "Stopping d10", Done: true},
	}}, nil)
	require.Equal(t, []string{"Starting d11|exit status 1", "Stopping d10|"}, ended)
	require.Empty(t, c.RunningJobs())

	c.setStatus(nil, errors.New("the status stream ended"))
	_, err := c.Projects()
	require.ErrorContains(t, err, "ended")
	c.setStatus(&serveStatus{Jobs: []jobState{{ID: 3, Title: "Stopping d10", Done: true}}}, nil)
	require.Len(t, ended, 2, "a reconnect repeats no notification")
}

// TestProxy checks that the window's requests reach ddev serve with the
// token, and that a state change without the page's header does not.
func TestProxy(t *testing.T) {
	var gotAuth, gotCookie, gotHost string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotCookie, gotHost = r.Header.Get("Authorization"), r.Header.Get("Cookie"), r.Host
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(backend.Close)
	u, err := url.Parse(backend.URL)
	require.NoError(t, err)
	c := &serveClient{base: u, token: "secret"}

	req := httptest.NewRequest("GET", "/api/events", nil)
	req.Host = "wails.localhost"
	req.Header.Set("Cookie", "a=b")
	rec := httptest.NewRecorder()
	c.Proxy().ServeHTTP(rec, req)
	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Equal(t, "Bearer secret", gotAuth)
	require.Empty(t, gotCookie)
	require.Equal(t, u.Host, gotHost, "ddev serve answers only to its own host")

	gotAuth = ""
	rec = httptest.NewRecorder()
	c.Proxy().ServeHTTP(rec, httptest.NewRequest("POST", "/api/actions/poweroff", nil))
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Empty(t, gotAuth, "the refused request did not reach ddev serve")
}

// TestServeClient runs a real `ddev serve --desktop`. It needs
// DDEV_BINARY, a ddev built from this branch.
func TestServeClient(t *testing.T) {
	bin := os.Getenv("DDEV_BINARY")
	if bin == "" {
		t.Skip("DDEV_BINARY is not set")
	}
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DDEV_NO_INSTRUMENTATION", "true")

	c, err := startServe(target{ddev: bin})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	require.Regexp(t, `^http://127\.0\.0\.1:\d+/\?token=[0-9a-f]{64}$`, c.URL())

	changed := c.Changed()
	go c.Follow()
	select {
	case <-changed:
	case <-time.After(30 * time.Second):
		t.Fatal("no status arrived")
	}
	require.NotEmpty(t, c.Version())
	// Without Docker the status carries an error instead of projects. With
	// it, running projects are listed even outside this configuration.
	if _, err := c.Projects(); err != nil {
		require.ErrorContains(t, err, "Docker")
	}

	require.ErrorContains(t, c.Post("/api/projects/nonexistent/actions/start"), "nonexistent")
	require.Error(t, c.Post("/api/actions/nothing"))
	require.Empty(t, c.RunningJobs(), "no job has started")

	// The window's page comes through the proxy without a token.
	rec := httptest.NewRecorder()
	c.Proxy().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<html")

	c.Close()
	select {
	case <-c.Exited():
	case <-time.After(10 * time.Second):
		t.Fatal("ddev serve did not stop when its stdin closed")
	}
}

func TestStartServeReportsOldDdev(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("needs a shell script")
	}
	old := t.TempDir() + "/ddev"
	require.NoError(t, os.WriteFile(old, []byte("#!/bin/sh\necho 'Error: unknown command \"serve\" for \"ddev\"'\nexit 1\n"), 0o755))
	_, err := startServe(target{ddev: old})
	require.ErrorContains(t, err, "needs a ddev with `ddev serve`")
	require.ErrorContains(t, err, `unknown command "serve"`)
}
