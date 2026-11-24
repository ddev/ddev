package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The app holds no DDEV code. It runs `ddev serve --desktop` from the one
// ddev it finds, shows that dashboard in its window, and drives the tray
// through the same HTTP API, so the window, the tray, and the terminal all
// use the same ddev.

// target is where the ddev that the app runs lives.
type target struct {
	// ddev is the ddev binary to run; it is "ddev" inside WSL2.
	ddev string
	// wsl is set when ddev runs inside WSL2, in distro, or in the default
	// distro when distro is empty.
	wsl    bool
	distro string
}

func (t target) String() string {
	if !t.wsl {
		return t.ddev
	}
	if t.distro == "" {
		return "ddev in WSL2"
	}
	return "ddev in WSL2 (" + t.distro + ")"
}

// chooseTarget picks the ddev to run. $DDEV_DESKTOP_MODE, native or wsl,
// overrides the choice. On Windows, a ddev.exe next to the app is a
// traditional Windows install; otherwise the app uses ddev in WSL2, where
// the installer's WSL2 choices put it, and finally a ddev.exe on PATH.
// wslDistro, from --wsl-distro, names the distro and implies WSL2.
func chooseTarget(wslDistro string) (target, error) {
	mode := os.Getenv("DDEV_DESKTOP_MODE")
	switch mode {
	case "", "native", "wsl":
	default:
		return target{}, fmt.Errorf("DDEV_DESKTOP_MODE must be native or wsl, not %q", mode)
	}
	if mode == "wsl" || wslDistro != "" {
		return target{ddev: "ddev", wsl: true, distro: wslDistro}, nil
	}
	if runtime.GOOS == "windows" && mode == "" && os.Getenv("DDEV_BINARY") == "" {
		if b := siblingDdev(); b != "" {
			return target{ddev: b}, nil
		}
		if wslHasDdev("") {
			return target{ddev: "ddev", wsl: true}, nil
		}
	}
	b, err := findDdev()
	if err != nil {
		return target{}, err
	}
	return target{ddev: b}, nil
}

func wslArgs(distro string, script string) []string {
	args := []string{}
	if distro != "" {
		args = append(args, "-d", distro)
	}
	// A login shell gives ddev the PATH that the user's terminal has.
	return append(args, "-e", "bash", "-lc", script)
}

func wslHasDdev(distro string) bool {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "wsl.exe", wslArgs(distro, "command -v ddev")...).Run() == nil
}

// project is what the tray shows of a project.
type project struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	URL     string `json:"url"`
	Mailpit string `json:"mailpit"`
	Up      bool   `json:"up"`
}

type jobState struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Project string `json:"project"`
	Done    bool   `json:"done"`
	Err     string `json:"err"`
}

// serveStatus is what the app reads of a status event.
type serveStatus struct {
	Projects []project  `json:"projects"`
	Jobs     []jobState `json:"jobs"`
	Error    string     `json:"error"`
	Version  string     `json:"version"`
}

// serveClient runs `ddev serve --desktop` and follows its status stream.
type serveClient struct {
	target target
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	exited chan struct{}
	launch string
	base   *url.URL
	token  string
	client *http.Client
	output *tail
	// closing is set once the app stops ddev serve itself.
	closing atomic.Bool

	// OnJobDone, when set before Follow, is called as each job ends.
	OnJobDone func(title, project, errText string)

	mu      sync.Mutex
	status  *serveStatus
	statErr error
	changed chan struct{}
	// jobs is whether each job seen was done, to notice the ends.
	jobs map[int]bool
}

var serveURL = regexp.MustCompile(`http://127\.0\.0\.1:\d+/\?token=[0-9a-f]+`)

// startServe starts `ddev serve --desktop` and waits for the URL it prints.
func startServe(t target) (*serveClient, error) {
	args := []string{"serve", "--no-open", "--desktop"}
	cmd := exec.Command(t.ddev, args...)
	if t.wsl {
		cmd = exec.Command("wsl.exe", wslArgs(t.distro, "ddev "+strings.Join(args, " "))...)
	}
	// ddev serve --desktop stops when its stdin closes, so it does not
	// outlive this app, inside WSL2 or not.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to run %s: %w", t, err)
	}
	out := &tail{}
	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			if u := serveURL.FindString(line); u != "" {
				select {
				case found <- u:
				default:
				}
				continue
			}
			out.add(line)
		}
	}()
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
		close(exited)
	}()
	select {
	case launch := <-found:
		u, err := url.Parse(launch)
		if err != nil {
			return nil, err
		}
		return &serveClient{
			target:  t,
			cmd:     cmd,
			stdin:   stdin,
			exited:  exited,
			launch:  launch,
			base:    &url.URL{Scheme: u.Scheme, Host: u.Host},
			token:   u.Query().Get("token"),
			client:  &http.Client{Timeout: 30 * time.Second},
			output:  out,
			changed: make(chan struct{}),
			jobs:    map[int]bool{},
		}, nil
	case <-exited:
		return nil, fmt.Errorf("`ddev serve --desktop` from %s stopped without printing a URL. This app needs a ddev with `ddev serve`.\n\n%s", t, out.String())
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("`ddev serve --desktop` from %s printed no URL within 2 minutes.\n\n%s", t, out.String())
	}
}

// URL is the launch URL, which carries the token.
func (c *serveClient) URL() string { return c.launch }

// Exited is closed when ddev serve has stopped.
func (c *serveClient) Exited() <-chan struct{} { return c.exited }

// Closing is whether the app stopped ddev serve itself.
func (c *serveClient) Closing() bool { return c.closing.Load() }

// Output is the last lines ddev serve printed.
func (c *serveClient) Output() string { return c.output.String() }

func (c *serveClient) authorize(h http.Header) {
	h.Set("Authorization", "Bearer "+c.token)
	h.Set(dashboardHeader, "1")
}

// dashboardHeader is sent by the page with every state change. A page from
// another origin cannot add it without a CORS preflight, which neither the
// proxy nor ddev serve answers.
const dashboardHeader = "X-DDEV-Dashboard"

// Proxy passes the window's requests to ddev serve with the token, so the
// window needs no token, cookie, or port of its own. A state change must
// still carry the page's header, which another page could not send.
func (c *serveClient) Proxy() http.Handler {
	rp := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(c.base)
			r.Out.Header.Del("Cookie")
			c.authorize(r.Out.Header)
		},
		// The status stream and the logs must arrive as they are written.
		FlushInterval: -1,
	}
	// Streams break whenever the window reloads or the app quits.
	if os.Getenv("DDEV_DESKTOP_DEBUG") != "true" {
		rp.ErrorLog = log.New(io.Discard, "", 0)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get(dashboardHeader) == "" {
			http.Error(w, "request refused", http.StatusForbidden)
			return
		}
		rp.ServeHTTP(w, r)
	})
}

// Follow reads the status stream until ddev serve stops, reconnecting when
// the stream breaks.
func (c *serveClient) Follow() {
	for {
		err := c.stream()
		select {
		case <-c.exited:
			c.setStatus(nil, errors.New("ddev serve stopped"))
			return
		default:
		}
		if err != nil {
			c.setStatus(nil, err)
		}
		time.Sleep(time.Second)
	}
}

func (c *serveClient) stream() error {
	req, err := http.NewRequest(http.MethodGet, c.base.String()+"/api/events", nil)
	if err != nil {
		return err
	}
	c.authorize(req.Header)
	// A stream never ends, so it gets no client timeout.
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("the status stream answered %s", res.Status)
	}
	return readStatusEvents(res.Body, func(st *serveStatus) { c.setStatus(st, nil) })
}

// readStatusEvents calls each with every status event in a stream.
func readStatusEvents(r io.Reader, each func(*serveStatus)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	isStatus := false
	for scanner.Scan() {
		line := scanner.Text()
		if e, ok := strings.CutPrefix(line, "event: "); ok {
			isStatus = e == "status"
			continue
		}
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || !isStatus {
			continue
		}
		var st serveStatus
		if err := json.Unmarshal([]byte(data), &st); err != nil {
			return err
		}
		each(&st)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return errors.New("the status stream ended")
}

func (c *serveClient) setStatus(st *serveStatus, err error) {
	var ended []jobState
	c.mu.Lock()
	if st != nil {
		first := c.status == nil && len(c.jobs) == 0
		for _, j := range st.Jobs {
			wasDone, seen := c.jobs[j.ID]
			// A job that started and ended between two events ends too.
			if j.Done && !wasDone && (seen || !first) {
				ended = append(ended, j)
			}
			c.jobs[j.ID] = j.Done
		}
		c.status, c.statErr = st, nil
	} else {
		c.status, c.statErr = nil, err
	}
	close(c.changed)
	c.changed = make(chan struct{})
	c.mu.Unlock()
	if c.OnJobDone != nil {
		for _, j := range ended {
			c.OnJobDone(j.Title, j.Project, j.Err)
		}
	}
}

// Changed is closed when the next status arrives.
func (c *serveClient) Changed() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.changed
}

// Projects returns the projects of the last status, or why there are none,
// such as Docker not running.
func (c *serveClient) Projects() ([]project, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case c.statErr != nil:
		return nil, c.statErr
	case c.status == nil:
		return nil, errors.New("connecting to ddev serve…")
	case c.status.Error != "":
		return nil, errors.New(c.status.Error)
	}
	return slices.Clone(c.status.Projects), nil
}

// Version is the version of the ddev that the app runs.
func (c *serveClient) Version() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status == nil {
		return ""
	}
	return c.status.Version
}

// RunningJobs returns the titles of the jobs that have not ended, which
// quitting would stop.
func (c *serveClient) RunningJobs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	if c.status != nil {
		for _, j := range c.status.Jobs {
			if !j.Done {
				out = append(out, j.Title)
			}
		}
	}
	return out
}

// Post sends a POST to ddev serve as the page would, so that an action
// started from the tray shows in the page's activity list.
func (c *serveClient) Post(path string) error {
	req, err := http.NewRequest(http.MethodPost, c.base.String()+path, nil)
	if err != nil {
		return err
	}
	c.authorize(req.Header)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return errors.New(strings.TrimSpace(string(b)))
	}
	return nil
}

// Close stops ddev serve by closing its stdin, which also stops the jobs it
// runs, then kills it if needed.
func (c *serveClient) Close() {
	c.closing.Store(true)
	_ = c.stdin.Close()
	select {
	case <-c.exited:
	case <-time.After(5 * time.Second):
		_ = c.cmd.Process.Kill()
	}
}

// tail keeps the last lines of output, for an error message.
type tail struct {
	mu    sync.Mutex
	lines []string
}

func (t *tail) add(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lines = append(t.lines, line)
	if len(t.lines) > 20 {
		t.lines = t.lines[len(t.lines)-20:]
	}
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.TrimSpace(strings.Join(t.lines, "\n"))
}
