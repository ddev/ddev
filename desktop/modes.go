package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/webui"
)

// mode is how the window reaches the dashboard.
type mode string

const (
	// modeEmbedded passes the window's requests to an in-process handler.
	modeEmbedded mode = "embedded"
	// modeLoopback serves the dashboard in-process on 127.0.0.1 with a
	// token, and the window loads its URL.
	modeLoopback mode = "loopback"
	// modeWSL runs `ddev serve` inside WSL2, and the window loads its URL
	// through WSL2's localhost forwarding.
	modeWSL mode = "wsl"
)

// chooseMode returns $DDEV_DESKTOP_MODE when set. WebView2 on Windows
// cannot stream a response, so there the window loads a URL instead: from
// WSL2 when ddev is installed there, which is where most Windows users run
// DDEV, or else from a server in this process.
func chooseMode() (mode, error) {
	switch m := mode(os.Getenv("DDEV_DESKTOP_MODE")); m {
	case modeEmbedded, modeLoopback, modeWSL:
		return m, nil
	case "":
	default:
		return "", fmt.Errorf("DDEV_DESKTOP_MODE must be embedded, loopback, or wsl, not %q", m)
	}
	if runtime.GOOS != "windows" {
		return modeEmbedded, nil
	}
	if wslHasDdev() {
		return modeWSL, nil
	}
	return modeLoopback, nil
}

func wslHasDdev() bool {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "wsl.exe", "-e", "bash", "-lc", "command -v ddev").Run() == nil
}

// backend is what the tray menu needs from the dashboard.
type backend interface {
	Projects() ([]webui.ProjectState, error)
	// Changed is closed when a job starts or ends; nil means poll only.
	Changed() <-chan struct{}
	Post(path string) error
}

var _ backend = (*webui.Server)(nil)

// remoteServer is a `ddev serve` this app started, inside WSL2 on Windows.
// The tray menu reaches it with the token, as a script would.
type remoteServer struct {
	cmd    *exec.Cmd
	exited chan struct{}
	stdin  io.WriteCloser
	launch string
	base   string
	token  string
	client *http.Client
}

var serveURL = regexp.MustCompile(`http://127\.0\.0\.1:\d+/\?token=[0-9a-f]+`)

// startRemote starts `ddev serve` and waits for the URL it prints.
// Elsewhere than Windows it runs ddevBinary, which tests this mode.
func startRemote(ddevBinary string) (*remoteServer, error) {
	args := []string{"serve", "--no-open", "--desktop"}
	cmd := exec.Command(ddevBinary, args...)
	if runtime.GOOS == "windows" {
		// A login shell gives ddev the PATH the user's terminal has.
		cmd = exec.Command("wsl.exe", "-e", "bash", "-lc", "ddev "+strings.Join(args, " "))
	}
	// ddev serve --desktop stops when its stdin closes, so it does not
	// outlive this app inside WSL2.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			if u := serveURL.FindString(scanner.Text()); u != "" {
				select {
				case found <- u:
				default:
				}
			}
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
		return &remoteServer{
			cmd:    cmd,
			exited: exited,
			stdin:  stdin,
			launch: launch,
			base:   u.Scheme + "://" + u.Host,
			token:  u.Query().Get("token"),
			client: &http.Client{Timeout: 30 * time.Second},
		}, nil
	case <-exited:
		return nil, errors.New("ddev serve exited without printing a URL")
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		return nil, errors.New("ddev serve printed no URL within 2 minutes")
	}
}

// URL is the launch URL, which carries the token.
func (r *remoteServer) URL() string { return r.launch }

func (r *remoteServer) request(method, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, r.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("X-DDEV-Dashboard", "1")
	return r.client.Do(req)
}

// Projects reads the first status event of the dashboard's stream.
func (r *remoteServer) Projects() ([]webui.ProjectState, error) {
	res, err := r.request(http.MethodGet, "/api/events")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	scanner := bufio.NewScanner(res.Body)
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
		var st struct {
			Projects []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				URL    string `json:"url"`
			} `json:"projects"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &st); err != nil {
			return nil, err
		}
		if st.Error != "" {
			return nil, errors.New(st.Error)
		}
		out := make([]webui.ProjectState, 0, len(st.Projects))
		for _, p := range st.Projects {
			out = append(out, webui.ProjectState{Name: p.Name, Status: p.Status, URL: p.URL})
		}
		return out, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("the status stream ended without a status")
}

func (r *remoteServer) Changed() <-chan struct{} { return nil }

func (r *remoteServer) Post(path string) error {
	res, err := r.request(http.MethodPost, path)
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

// Close stops `ddev serve` by closing its stdin, then kills it if needed.
func (r *remoteServer) Close() {
	_ = r.stdin.Close()
	select {
	case <-r.exited:
	case <-time.After(5 * time.Second):
		_ = r.cmd.Process.Kill()
	}
}
