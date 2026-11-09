package webui

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
)

// ProjectState is what a tray menu shows of a project.
type ProjectState struct {
	Name   string
	Status string
	URL    string
}

// Projects lists the projects, or returns why they cannot be listed, such as
// Docker not running.
func (s *Server) Projects() ([]ProjectState, error) {
	st := loadStatus()
	if st.Error != "" {
		return nil, errors.New(st.Error)
	}
	out := make([]ProjectState, 0, len(st.Projects))
	for _, p := range st.Projects {
		out = append(out, ProjectState{Name: p.Name, Status: p.Status, URL: p.URL})
	}
	return out, nil
}

// Changed is closed when a job starts or ends.
func (s *Server) Changed() <-chan struct{} {
	return s.changedCh()
}

// Post sends a POST through the handler as the page would, so that an action
// started from a tray menu shows in the page's activity list. A server from
// New gets the token, as a script would send it.
func (s *Server) Post(path string) error {
	r, err := http.NewRequest(http.MethodPost, path, http.NoBody)
	if err != nil {
		return err
	}
	r.Header.Set(dashboardHeader, "1")
	if s.listener != nil {
		r.Host = s.listener.Addr().String()
		r.Header.Set("Authorization", "Bearer "+s.token)
	}
	w := &postRecorder{header: http.Header{}, code: http.StatusOK}
	s.Handler().ServeHTTP(w, r)
	if w.code >= 400 {
		return errors.New(strings.TrimSpace(w.body.String()))
	}
	return nil
}

// postRecorder keeps the status and body of a response to Post.
type postRecorder struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (w *postRecorder) Header() http.Header         { return w.header }
func (w *postRecorder) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *postRecorder) WriteHeader(code int)        { w.code = code }
