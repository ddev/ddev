package webui

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
// started from a tray menu shows in the page's activity list.
func (s *Server) Post(path string) error {
	r := httptest.NewRequest(http.MethodPost, path, nil)
	r.Header.Set(dashboardHeader, "1")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code >= 400 {
		return errors.New(strings.TrimSpace(w.Body.String()))
	}
	return nil
}
