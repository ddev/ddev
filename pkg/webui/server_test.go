package webui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestMain lets the test binary stand in for ddev: with WEBUI_FAKE_DDEV set
// it prints its arguments and exits with the status that variable names.
func TestMain(m *testing.M) {
	if code := os.Getenv("WEBUI_FAKE_DDEV"); code != "" {
		fmt.Println("args:", strings.Join(os.Args[1:], " "))
		wd, _ := os.Getwd()
		fmt.Println("dir:", wd)
		fmt.Println("CLICOLOR_FORCE:", os.Getenv("CLICOLOR_FORCE"))
		fmt.Fprintln(os.Stderr, "\x1b[31mcolored\x1b[0m stderr\x1b[K")
		fmt.Println("progress 10%\rprogress 100%")
		if code == "sleep" {
			time.Sleep(10 * time.Second)
		}
		if code != "0" {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func newTestServer(t *testing.T) *Server {
	s, err := New(0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.listener.Close() })
	return s
}

func (s *Server) host() string {
	return s.listener.Addr().String()
}

func request(s *Server, method, target string, mod func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Host = s.host()
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func withCookie(s *Server) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: s.cookieName, Value: s.token}) }
}

// A browser sends a host's cookies to every port, so one shared name let a
// second `ddev gui --browser` log the first one's tab out.
func TestCookiePerPort(t *testing.T) {
	a, b := newTestServer(t), newTestServer(t)
	require.NotEqual(t, a.cookieName, b.cookieName)

	for _, s := range []*Server{a, b} {
		_, port, err := net.SplitHostPort(s.host())
		require.NoError(t, err)
		require.Equal(t, cookiePrefix+port, s.cookieName)
		rec := request(s, "GET", "/?token="+s.token, nil)
		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		require.Equal(t, s.cookieName, cookies[0].Name)
	}

	both := func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: a.cookieName, Value: a.token})
		r.AddCookie(&http.Cookie{Name: b.cookieName, Value: b.token})
	}
	require.Equal(t, http.StatusOK, request(a, "GET", "/", both).Code)
	require.Equal(t, http.StatusOK, request(b, "GET", "/", both).Code)
	require.Equal(t, http.StatusUnauthorized, request(a, "GET", "/", withCookie(b)).Code)
}

func TestSecurity(t *testing.T) {
	s := newTestServer(t)

	rec := request(s, "GET", "/", func(r *http.Request) { r.Host = "evil.example.com" })
	require.Equal(t, http.StatusForbidden, rec.Code, "a rebinding Host must be refused")

	rec = request(s, "GET", "/", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = request(s, "GET", "/?token=wrong", nil)
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = request(s, "GET", "/?token="+s.token, nil)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "/", rec.Header().Get("Location"))
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, s.token, cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)

	rec = request(s, "GET", "/", withCookie(s))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<title>DDEV</title>")
	require.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")

	rec = request(s, "GET", "/app.js", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+s.token) })
	require.Equal(t, http.StatusOK, rec.Code)

	rec = request(s, "GET", "/app.js", func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") })
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// Cookie-authenticated state changes must come from the dashboard's own
	// origin. An unknown action reaching the router shows the check passed.
	rec = request(s, "POST", "/api/actions/nothing", withCookie(s))
	require.Equal(t, http.StatusForbidden, rec.Code, "no Origin or Referer")

	rec = request(s, "POST", "/api/actions/nothing", func(r *http.Request) {
		withCookie(s)(r)
		r.Header.Set("Origin", "https://evil.example.com")
	})
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = request(s, "POST", "/api/actions/nothing", func(r *http.Request) {
		withCookie(s)(r)
		r.Header.Set("Origin", "http://"+s.host())
	})
	require.Equal(t, http.StatusNotFound, rec.Code)

	rec = request(s, "POST", "/api/actions/nothing", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+s.token)
	})
	require.Equal(t, http.StatusNotFound, rec.Code, "a bearer token needs no Origin")
}

// fakeDdev points the server's subcommands at the test binary.
func fakeDdev(t *testing.T, s *Server, exitCode string) {
	bin, err := os.Executable()
	require.NoError(t, err)
	s.ddevBinary = bin
	s.baseEnv = append(os.Environ(), "WEBUI_FAKE_DDEV="+exitCode)
}

// readSSE reads events from a stream until the `end` event.
func readSSE(t *testing.T, url string, header http.Header) (events []string) {
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header = header
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))
	scanner := bufio.NewScanner(res.Body)
	event := ""
	for scanner.Scan() {
		line := scanner.Text()
		if e, ok := strings.CutPrefix(line, "event: "); ok {
			event = e
		}
		if d, ok := strings.CutPrefix(line, "data: "); ok {
			events = append(events, event+" "+d)
			if event == "end" {
				return events
			}
		}
	}
	t.Fatalf("stream ended without an end event: %v", events)
	return nil
}

func TestJobRunsAndStreams(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "0")
	go func() { _ = s.Serve() }()
	t.Cleanup(func() { _ = s.http.Close() })

	auth := http.Header{"Authorization": {"Bearer " + s.token}}
	req, err := http.NewRequest("POST", "http://"+s.host()+"/api/actions/poweroff", nil)
	require.NoError(t, err)
	req.Header = auth
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()
	require.Equal(t, http.StatusAccepted, res.StatusCode)

	events := readSSE(t, "http://"+s.host()+"/api/jobs/1", auth)
	all := strings.Join(events, "\n")
	require.Contains(t, all, `"$ ddev poweroff"`)
	require.Contains(t, all, `"args: poweroff"`)
	require.Contains(t, all, `"CLICOLOR_FORCE: 1"`, "ddev colors its output for the page")
	require.Contains(t, all, `"\u001b[31mcolored\u001b[0m stderr"`, "colors stay, other sequences go")
	require.Contains(t, all, `"progress 100%"`, "only the text after a carriage return is kept")
	require.NotContains(t, all, "progress 10%")
	require.Equal(t, `end {"err":""}`, events[len(events)-1])
}

// TestShutdownEndsStreams checks that Ctrl+C does not wait for an open
// browser tab's status stream until Shutdown's timeout.
func TestShutdownEndsStreams(t *testing.T) {
	s := newTestServer(t)
	served := make(chan error, 1)
	go func() { served <- s.Serve() }()

	req, err := http.NewRequest("GET", "http://"+s.host()+"/api/events", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	_, err = bufio.NewReader(res.Body).ReadString('\n')
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	require.NoError(t, s.Shutdown(ctx))
	require.Less(t, time.Since(start), 2*time.Second)
	require.NoError(t, <-served)
}

func TestJobFailureAndBusy(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "1")
	j, err := s.jobs.start("Failing", "p1", s.ddevCommand(t.Context(), "", "start", "p1"))
	require.NoError(t, err)
	require.Eventually(t, func() bool { return j.summary().Done }, 10*time.Second, 50*time.Millisecond)
	require.NotEmpty(t, j.summary().Err)

	fakeDdev(t, s, "sleep")
	long := s.ddevCommand(t.Context(), "", "start", "p2")
	_, err = s.jobs.start("Long", "p2", long)
	require.NoError(t, err)
	t.Cleanup(func() { _ = long.Process.Kill() })

	_, err = s.jobs.start("Same project", "p2", s.ddevCommand(t.Context(), ""))
	require.ErrorAs(t, err, &errBusy{})
	_, err = s.jobs.start("Global", "", s.ddevCommand(t.Context(), ""))
	require.ErrorAs(t, err, &errBusy{}, "a global job waits for every project's jobs")

	other := s.ddevCommand(t.Context(), "", "start", "p3")
	_, err = s.jobs.start("Other project", "p3", other)
	require.NoError(t, err)
	t.Cleanup(func() { _ = other.Process.Kill() })
}

func TestCleanLine(t *testing.T) {
	require.Equal(t, "done", cleanLine("50%\rdone\r\n"))
	// Colors stay; cursor movement, hyperlinks and cut sequences go.
	require.Equal(t, "\x1b[31mred\x1b[0m text", cleanLine("\x1b[31mred\x1b[0m text"))
	require.Equal(t, "\x1b[1;32mok\x1b[0m", cleanLine("\x1b[2K\x1b[1;32mok\x1b[0m\x1b[1A"))
	require.Equal(t, "docs", cleanLine("\x1b]8;;https://docs.ddev.com\x1b\\docs\x1b]8;;\x1b\\"))
	require.Equal(t, "cut ", cleanLine("cut \x1b[3"))
	require.Equal(t, "lone ", cleanLine("lone \x1b"))
	require.Equal(t, "plain", cleanLine("plain"))
}

func TestReadOutput(t *testing.T) {
	pr, pw := io.Pipe()
	var lines []string
	partials := make(chan string)
	go readOutput(pr, func(l string) { lines = append(lines, l) }, func(p string) { partials <- p })

	write := func(s string) {
		go func() { _, _ = pw.Write([]byte(s)) }()
	}
	write("Building project images...")
	require.Equal(t, "Building project images...", <-partials)
	write("..")
	require.Equal(t, "Building project images.....", <-partials)
	write(" built\nStarting \xc3")
	require.Equal(t, "Starting ", <-partials, "half a rune waits for the rest")
	write("\xa9\r\x1b[1m50%")
	require.Equal(t, "\x1b[1m50%", <-partials)
	_ = pw.Close()
	require.Equal(t, "", <-partials)
	require.Equal(t, []string{"Building project images..... built", "\x1b[1m50%"}, lines)
}

func TestEditorURL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	require.Equal(t, "vscode://file/home/me/my%20site?windowId=_blank", editorURL("vscode", "/home/me/my site"))
	require.Equal(t, "phpstorm://open?file=/home/me/a%26b", editorURL("phpstorm", "/home/me/a&b"))
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	require.Equal(t, "vscode://vscode-remote/wsl+Ubuntu/home/me/site?windowId=_blank", editorURL("vscode", "/home/me/site"))
}
