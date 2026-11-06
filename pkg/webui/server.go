// Package webui serves an experimental browser dashboard for DDEV projects,
// `ddev serve`. It reads projects through pkg/ddevapp, runs actions as ddev
// subcommands the way `ddev tui` does, and streams their output and project
// status to the browser with Server-Sent Events.
//
// It listens on 127.0.0.1 only, and every request must carry the token from
// the launch URL, since any page the user visits could otherwise reach it.
package webui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

//go:embed assets
var assets embed.FS

// cookieName holds the token once the launch URL has been visited, so it
// does not stay in the address bar or history.
const cookieName = "ddev_serve_token"

// Server is the dashboard's HTTP server.
type Server struct {
	token string
	// hosts are the Host header values the server answers to, which stops
	// DNS rebinding from reaching it through another name.
	hosts map[string]bool
	// origins are the Origin header values a state-changing request may
	// carry.
	origins  map[string]bool
	listener net.Listener
	http     *http.Server
	jobs     *jobs
	// baseEnv is the environment from before any project was described.
	// Describe sets project variables on this process, which must not leak
	// into a subcommand for another project.
	baseEnv []string
	// ddevBinary is the ddev binary subcommands run with.
	ddevBinary string

	// changed is closed and replaced whenever a job starts or ends, so the
	// status streams reload at once instead of on their next tick.
	changedMu sync.Mutex
	changed   chan struct{}
}

// New listens on 127.0.0.1:port, or a free port when port is 0.
func New(port int) (*Server, error) {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		_ = l.Close()
		return nil, err
	}
	bin, err := os.Executable()
	if err != nil {
		_ = l.Close()
		return nil, err
	}
	s := &Server{
		token:      hex.EncodeToString(b),
		listener:   l,
		baseEnv:    os.Environ(),
		ddevBinary: bin,
		changed:    make(chan struct{}),
	}
	s.jobs = newJobs(s.notifyChanged)
	actual := l.Addr().(*net.TCPAddr).Port
	s.hosts, s.origins = map[string]bool{}, map[string]bool{}
	for _, h := range []string{"127.0.0.1", "localhost"} {
		hp := net.JoinHostPort(h, strconv.Itoa(actual))
		s.hosts[hp] = true
		s.origins["http://"+hp] = true
	}
	s.http = &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return s, nil
}

// URL is the launch URL, which carries the token.
func (s *Server) URL() string {
	return fmt.Sprintf("http://%s/?token=%s", s.listener.Addr().String(), s.token)
}

// Serve serves until Shutdown.
func (s *Server) Serve() error {
	err := s.http.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown stops the server and the subcommands it started.
func (s *Server) Shutdown(ctx context.Context) error {
	s.jobs.killAll()
	return s.http.Shutdown(ctx)
}

func (s *Server) notifyChanged() {
	s.changedMu.Lock()
	defer s.changedMu.Unlock()
	close(s.changed)
	s.changed = make(chan struct{})
}

func (s *Server) changedCh() <-chan struct{} {
	s.changedMu.Lock()
	defer s.changedMu.Unlock()
	return s.changed
}

// Handler returns the routes wrapped in the security checks.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "assets")
	mux.Handle("GET /", http.FileServerFS(static))

	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("GET /api/projects/{name}", s.handleDetail)
	mux.HandleFunc("POST /api/projects/{name}/actions/{action}", s.handleProjectAction)
	mux.HandleFunc("POST /api/projects/{name}/open/{target}", s.handleOpen)
	mux.HandleFunc("GET /api/projects/{name}/logs", s.handleLogs)
	mux.HandleFunc("POST /api/projects", s.handleAddProject)
	mux.HandleFunc("POST /api/actions/{action}", s.handleGlobalAction)
	mux.HandleFunc("GET /api/jobs/{id}", s.handleJobStream)
	mux.HandleFunc("GET /api/project-types", s.handleProjectTypes)
	mux.HandleFunc("GET /api/addons/registry", s.handleAddonRegistry)
	return s.secure(mux)
}

// secure rejects requests to an unexpected host, without the token, or,
// for a state change, from another origin. The launch URL's token is moved
// to a cookie and the browser sent back to /.
func (s *Server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")

		if !s.hosts[r.Host] {
			http.Error(w, "unexpected Host header", http.StatusForbidden)
			return
		}
		if q := r.URL.Query().Get("token"); q != "" && r.Method == http.MethodGet && r.URL.Path == "/" {
			if !s.tokenMatches(q) {
				http.Error(w, "invalid token", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		bearer := false
		if auth := r.Header.Get("Authorization"); auth != "" {
			const prefix = "Bearer "
			if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix || !s.tokenMatches(auth[len(prefix):]) {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			bearer = true
		} else if c, err := r.Cookie(cookieName); err != nil || !s.tokenMatches(c.Value) {
			http.Error(w, "Open the URL that `ddev serve` printed, which includes the access token.", http.StatusUnauthorized)
			return
		}
		// A cookie is sent with cross-site requests in some cases, so a
		// state change through it must come from this origin.
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !bearer && !s.sameOrigin(r) {
			http.Error(w, "cross-origin request refused", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) tokenMatches(t string) bool {
	return subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1
}

func (s *Server) sameOrigin(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" {
		return s.origins[o]
	}
	ref, err := url.Parse(r.Header.Get("Referer"))
	return err == nil && s.origins[ref.Scheme+"://"+ref.Host]
}
