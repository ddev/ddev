package webui

import (
	"cmp"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/version"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// refreshInterval is how often the status stream reloads the project list
// when no job has changed it.
const refreshInterval = 5 * time.Second

// handleEvents streams the project list as `status` events, and what the
// GUI app asks its window to show as `show` events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	shown := s.showCh()
	for {
		changed := s.changedCh()
		st := loadStatus()
		st.Jobs = s.jobs.summaries()
		st.Desktop = s.desktop
		st.Version = versionconstants.DdevVersion
		sse.send("status", st)
		refresh := time.After(refreshInterval)
	wait:
		select {
		case <-changed:
		case <-refresh:
		case <-shown:
			shown = s.showCh()
			sse.send("show", s.shownTarget())
			goto wait
		case <-r.Context().Done():
			return
		}
	}
}

// handleShow asks the window to show About DDEV or a project, for the GUI
// app's tray. Pointing the window at a new URL instead would leave WebView2
// black when the window is shown again.
func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target != "about" && !strings.HasPrefix(target, "project=") {
		http.Error(w, "unknown target", http.StatusBadRequest)
		return
	}
	s.requestShow(target)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	d, err := loadDetail(r.PathValue("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// formArg returns a form value to pass to ddev, refusing one that ddev
// would read as a flag.
func formArg(r *http.Request, key string) (string, bool) {
	v := strings.TrimSpace(r.FormValue(key))
	return v, !strings.HasPrefix(v, "-")
}

func (s *Server) handleProjectAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	root, err := appRoot(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	bad := func(msg string) { http.Error(w, msg, http.StatusBadRequest) }
	switch action := r.PathValue("action"); action {
	case "start", "stop", "restart":
		verb := map[string]string{"start": "Start", "stop": "Stop", "restart": "Restart"}[action]
		s.startJob(w, verb+" "+name, name, "", action, name)
	case "delete":
		args := []string{"delete", "-y"}
		title := "Delete " + name
		if r.FormValue("omit_snapshot") == "true" {
			args = append(args, "--omit-snapshot")
			title += " without a snapshot"
		}
		s.startJob(w, title, name, "", append(args, name)...)
	case "snapshot":
		s.startJob(w, "Make a snapshot of "+name, name, root, "snapshot")
	case "restore":
		snap, ok := formArg(r, "snapshot")
		if !ok {
			bad("invalid snapshot")
			return
		}
		if snap == "" {
			s.startJob(w, "Restore the latest snapshot of "+name, name, root, "snapshot", "restore", "--latest")
			return
		}
		s.startJob(w, "Restore snapshot "+filepath.Base(snap)+" of "+name, name, root, "snapshot", "restore", snap)
	case "addon-get":
		repo, ok := formArg(r, "repo")
		if !ok || repo == "" {
			bad("an add-on repository such as ddev/ddev-redis is required")
			return
		}
		s.startJob(w, "Install "+repo+" in "+name, name, root, "add-on", "get", repo)
	case "addon-remove":
		addon, ok := formArg(r, "addon")
		if !ok || addon == "" {
			bad("an add-on name is required")
			return
		}
		s.startJob(w, "Remove "+addon+" from "+name, name, root, "add-on", "remove", addon)
	case "addon-update":
		s.startJob(w, "Update the add-ons of "+name, name, root, "add-on", "update", "--yes")
	case "xdebug-on", "xdebug-off":
		state := strings.TrimPrefix(action, "xdebug-")
		s.startJob(w, "Turn Xdebug "+state+" in "+name, name, root, "xdebug", state)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleGlobalAction(w http.ResponseWriter, r *http.Request) {
	switch r.PathValue("action") {
	case "start-all":
		s.startJob(w, "Start all projects", "", "", "start", "--all")
	case "stop-all":
		s.startJob(w, "Stop all projects", "", "", "stop", "--all")
	case "poweroff":
		s.startJob(w, "Power off DDEV", "", "", "poweroff")
	case "start", "stop", "restart":
		// Several projects in one ddev command, which conflicts with every
		// project's jobs, as start-all does.
		_ = r.ParseForm()
		names := slices.Compact(slices.Sorted(slices.Values(r.PostForm["project"])))
		if len(names) == 0 {
			http.Error(w, "no projects given", http.StatusBadRequest)
			return
		}
		for _, name := range names {
			if _, err := appRoot(name); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
		}
		verb := map[string]string{"start": "Start", "stop": "Stop", "restart": "Restart"}[r.PathValue("action")]
		s.startJob(w, verb+" "+strings.Join(names, ", "), "", "", append([]string{r.PathValue("action")}, names...)...)
	default:
		http.NotFound(w, r)
	}
}

// handleAddProject runs `ddev config` in an existing directory.
func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.FormValue("dir"))
	if rest, ok := strings.CutPrefix(dir, "~"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, rest)
		}
	}
	if !filepath.IsAbs(dir) {
		http.Error(w, "the project directory must be an absolute path", http.StatusBadRequest)
		return
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		http.Error(w, "no such directory: "+dir, http.StatusBadRequest)
		return
	}
	args := []string{"config"}
	if t := r.FormValue("type"); t != "" {
		if !ddevapp.IsValidAppType(t) {
			http.Error(w, "unknown project type: "+t, http.StatusBadRequest)
			return
		}
		args = append(args, "--project-type="+t)
	}
	if n := strings.TrimSpace(r.FormValue("name")); n != "" {
		args = append(args, "--project-name="+n)
	}
	if d := strings.TrimSpace(r.FormValue("docroot")); d != "" {
		args = append(args, "--docroot="+d)
	}
	s.startJob(w, "Configure "+dir, filepath.Base(dir), dir, args...)
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	root, err := appRoot(r.PathValue("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var target string
	switch r.PathValue("target") {
	case "folder":
		target = root
	case "vscode", "phpstorm":
		target = editorURL(r.PathValue("target"), root)
	default:
		http.NotFound(w, r)
		return
	}
	if err := openTarget(target); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleOpenURL opens an http or https link in the default browser, since a
// desktop webview opens no new windows.
func (s *Server) handleOpenURL(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.FormValue("url"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "only http and https links can be opened", http.StatusBadRequest)
		return
	}
	if err := openTarget(u.String()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleLogs follows a service's logs as `lines` events until the browser
// leaves.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	root, err := appRoot(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	service, ok := formArg(r, "service")
	if !ok {
		http.Error(w, "invalid service", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	c := s.ddevCommand(ctx, root, "logs", "-f", "--tail", "500", "-s", cmp.Or(service, "web"))
	pr, pw, err := os.Pipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Stdout, c.Stderr = pw, pw
	if err := c.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = pw.Close()
	sse, ok := newSSE(w)
	if !ok {
		_ = pr.Close()
		return
	}
	streamLines(ctx, sse, pr, c.Wait)
}

// streamLines sends what pr reads as `lines` events until it ends, then an
// `end` event with wait's error. When ctx ends first, it closes pr and waits.
func streamLines(ctx context.Context, sse *sse, pr io.ReadCloser, wait func() error) {
	lines := make(chan string, 100)
	go func() {
		scanLines(pr, func(l string) {
			select {
			case lines <- l:
			case <-ctx.Done():
			}
		})
		close(lines)
	}()
	// Lines that arrive together go out in one event.
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				err := wait()
				errText := ""
				if err != nil && ctx.Err() == nil {
					errText = err.Error()
				}
				sse.send("end", map[string]string{"err": errText})
				return
			}
			batch := []string{l}
		more:
			for len(batch) < 500 {
				select {
				case l, ok := <-lines:
					if !ok {
						break more
					}
					batch = append(batch, l)
				default:
					break more
				}
			}
			sse.send("lines", batch)
		case <-ctx.Done():
			_ = pr.Close()
			_ = wait()
			return
		}
	}
}

// about is what the About dialog shows.
type about struct {
	// App is the GUI app's version, empty in a browser.
	App string `json:"app,omitempty"`
	// Binary is the ddev that serves the page.
	Binary string `json:"binary"`
	// Version is `ddev version -j`'s raw output, and Error why part of it
	// is missing, such as Docker being unreachable.
	Version map[string]string `json:"version"`
	Error   string            `json:"error,omitempty"`
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	info, err := version.GetVersionInfo()
	a := about{App: s.appVersion, Binary: s.ddevBinary, Version: info}
	if err != nil {
		a.Error = err.Error()
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleProjectTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ddevapp.GetValidAppTypes())
}

// registryAddon is an add-on the registry at addons.ddev.com lists.
type registryAddon struct {
	Repo        string `json:"repo"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

func (s *Server) handleAddonRegistry(w http.ResponseWriter, _ *http.Request) {
	addons, err := ddevapp.ListAvailableAddonsFromRegistry()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]registryAddon, 0, len(addons))
	for _, a := range addons {
		out = append(out, registryAddon{Repo: a.User + "/" + a.Repo, Title: a.Title, Description: a.Description, Type: a.Type})
	}
	slices.SortFunc(out, func(a, b registryAddon) int {
		// Official add-ons first.
		if (a.Type == "official") != (b.Type == "official") {
			if a.Type == "official" {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Repo, b.Repo)
	})
	writeJSON(w, http.StatusOK, out)
}
