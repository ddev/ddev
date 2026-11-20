package webui

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
)

// refreshInterval is how often the status stream reloads the project list
// when no job has changed it.
const refreshInterval = 5 * time.Second

// handleEvents streams the project list as `status` events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	sse, ok := newSSE(w)
	if !ok {
		return
	}
	for {
		changed := s.changedCh()
		st := loadStatus()
		st.Jobs = s.jobs.summaries()
		st.Desktop = s.embedded || s.desktop
		sse.send("status", st)
		select {
		case <-changed:
		case <-time.After(refreshInterval):
		case <-r.Context().Done():
			return
		}
	}
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
		verb := map[string]string{"start": "Starting", "stop": "Stopping", "restart": "Restarting"}[action]
		s.startJob(w, verb+" "+name, name, "", action, name)
	case "delete":
		args := []string{"delete", "-y"}
		title := "Deleting " + name
		if r.FormValue("omit_snapshot") == "true" {
			args = append(args, "--omit-snapshot")
			title += " without a snapshot"
		}
		s.startJob(w, title, name, "", append(args, name)...)
	case "snapshot":
		s.startJob(w, "Making a snapshot of "+name, name, root, "snapshot")
	case "restore":
		snap, ok := formArg(r, "snapshot")
		if !ok {
			bad("invalid snapshot")
			return
		}
		if snap == "" {
			s.startJob(w, "Restoring the latest snapshot of "+name, name, root, "snapshot", "restore", "--latest")
			return
		}
		s.startJob(w, "Restoring snapshot "+filepath.Base(snap)+" of "+name, name, root, "snapshot", "restore", snap)
	case "addon-get":
		repo, ok := formArg(r, "repo")
		if !ok || repo == "" {
			bad("an add-on repository such as ddev/ddev-redis is required")
			return
		}
		s.startJob(w, "Installing "+repo+" in "+name, name, root, "add-on", "get", repo)
	case "addon-remove":
		addon, ok := formArg(r, "addon")
		if !ok || addon == "" {
			bad("an add-on name is required")
			return
		}
		s.startJob(w, "Removing "+addon+" from "+name, name, root, "add-on", "remove", addon)
	case "addon-update":
		s.startJob(w, "Updating the add-ons of "+name, name, root, "add-on", "update", "--yes")
	case "xdebug-on", "xdebug-off":
		state := strings.TrimPrefix(action, "xdebug-")
		s.startJob(w, "Turning Xdebug "+state+" in "+name, name, root, "xdebug", state)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleGlobalAction(w http.ResponseWriter, r *http.Request) {
	switch r.PathValue("action") {
	case "start-all":
		s.startJob(w, "Starting all projects", "", "", "start", "--all")
	case "stop-all":
		s.startJob(w, "Stopping all projects", "", "", "stop", "--all")
	case "poweroff":
		s.startJob(w, "Powering off DDEV", "", "", "poweroff")
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
	s.startJob(w, "Configuring "+dir, filepath.Base(dir), dir, args...)
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
				err := c.Wait()
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
			_ = c.Wait()
			return
		}
	}
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
