package webui

import (
	"context"
	"io"
	"net"
	"net/http"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// sharedService is a container that all projects share.
type sharedService struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Image  string `json:"image,omitempty"`
	// Health is the last health check's output, which says why a
	// container is unhealthy.
	Health string `json:"health,omitempty"`
	// Errors are the router's Traefik configuration errors.
	Errors string `json:"errors,omitempty"`
	// Dashboard is the router's Traefik dashboard.
	Dashboard string `json:"dashboard,omitempty"`
	Docs      string `json:"docs"`
}

var sharedServices = []sharedService{
	{Name: "ddev-router", Label: "Router", Docs: "users/extend/traefik-router/"},
	{Name: ddevapp.SSHAuthName, Label: "SSH agent", Docs: "users/usage/commands/#auth-ssh"},
}

// findShared finds a shared container the way `ddev logs -s` does.
func findShared(name string) (*container.Summary, error) {
	return dockerutil.FindContainerByLabels(map[string]string{
		"com.docker.compose.service": name,
		"com.docker.compose.oneoff":  "False",
	})
}

func (s *Server) handleShared(w http.ResponseWriter, _ *http.Request) {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	if _, err := dockerutil.GetDockerVersion(); err != nil {
		http.Error(w, "Docker is not reachable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	out := []sharedService{}
	for _, svc := range sharedServices {
		c, err := findShared(svc.Name)
		if err != nil || c == nil {
			svc.Status = ddevapp.SiteStopped
		} else {
			svc.Status, svc.Health = dockerutil.GetContainerHealth(c)
			svc.Image = c.Image
		}
		if svc.Name == "ddev-router" && c != nil {
			svc.Errors = ddevapp.GetRouterConfigErrors()
			if ip, err := dockerutil.GetDockerIP(); err == nil {
				svc.Dashboard = "http://" + net.JoinHostPort(ip, globalconfig.DdevGlobalConfig.TraefikMonitorPort)
			}
		}
		out = append(out, svc)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleSharedLogs follows a shared container's logs as `lines` events.
func (s *Server) handleSharedLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name != "ddev-router" && name != ddevapp.SSHAuthName {
		http.NotFound(w, r)
		return
	}
	c, err := findShared(name)
	if err != nil || c == nil {
		http.Error(w, name+" is not running", http.StatusNotFound)
		return
	}
	_, apiClient, err := dockerutil.GetDockerClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	rc, err := apiClient.ContainerLogs(ctx, c.ID, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "500"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, rc)
		_ = rc.Close()
		_ = pw.Close()
		done <- err
	}()
	sse, ok := newSSE(w)
	if !ok {
		_ = pr.Close()
		return
	}
	streamLines(ctx, sse, pr, func() error {
		if err := <-done; err != nil && ctx.Err() == nil {
			return err
		}
		return nil
	})
}
