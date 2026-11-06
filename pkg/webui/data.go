package webui

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
)

// ddevMu serializes reads of ddevapp, whose project list and environment
// are process-wide.
var ddevMu sync.Mutex

// projectSummary is a row of the project list.
type projectSummary struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`
	AppRoot string `json:"approot"`
}

// status is what the status stream sends.
type status struct {
	Projects []projectSummary `json:"projects"`
	Router   string           `json:"router"`
	Jobs     []jobSummary     `json:"jobs"`
	// Error is set when Docker cannot be reached or the list cannot be read.
	Error string `json:"error,omitempty"`
}

// loadStatus lists the projects the way `ddev list` does, with the router's
// health.
func loadStatus() status {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	st := status{Projects: []projectSummary{}}
	if err := globalconfig.ReadProjectList(); err != nil {
		st.Error = err.Error()
		return st
	}
	if _, err := dockerutil.GetDockerVersion(); err != nil {
		st.Error = "Docker is not reachable: " + err.Error()
		return st
	}
	defer dockerutil.CacheContainers()()
	apps, err := ddevapp.GetProjects(false)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	routerUsed := false
	for _, app := range apps {
		p := projectSummary{Name: app.GetName(), Type: app.GetType(), AppRoot: app.GetAppRoot()}
		p.Status, _ = app.SiteStatus()
		if p.Status == ddevapp.SiteRunning {
			p.URL = app.GetPrimaryURL()
		}
		st.Projects = append(st.Projects, p)
		routerUsed = routerUsed || !ddevapp.IsRouterDisabled(app)
	}
	slices.SortFunc(st.Projects, func(a, b projectSummary) int { return strings.Compare(a.Name, b.Name) })
	st.Router = "disabled"
	if routerUsed {
		st.Router, _ = ddevapp.GetRouterStatus()
	}
	return st
}

// addonInfo is an installed add-on.
type addonInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
}

// snapshotInfo is a snapshot `ddev snapshot restore` can restore.
type snapshotInfo struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Created string `json:"created"`
	Size    int64  `json:"size"`
}

// projectDetail is `ddev describe -j` with the add-ons and snapshots.
type projectDetail struct {
	Describe  map[string]any `json:"describe"`
	Addons    []addonInfo    `json:"addons"`
	Snapshots []snapshotInfo `json:"snapshots"`
	// Xdebug is "on" or "off" for a running project, and empty otherwise.
	Xdebug string `json:"xdebug,omitempty"`
}

// findApp returns the registered project called name.
func findApp(name string) (*ddevapp.DdevApp, error) {
	if err := globalconfig.ReadProjectList(); err != nil {
		return nil, err
	}
	p := globalconfig.GetProject(name)
	if p == nil {
		return nil, fmt.Errorf("no project named %q", name)
	}
	if _, err := os.Stat(p.AppRoot); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("project directory no longer exists: %s", p.AppRoot)
	}
	return ddevapp.NewApp(p.AppRoot, true)
}

// appRoot returns the directory of the registered project called name.
func appRoot(name string) (string, error) {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	app, err := findApp(name)
	if err != nil {
		return "", err
	}
	return app.GetAppRoot(), nil
}

func loadDetail(name string) (projectDetail, error) {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	app, err := findApp(name)
	if err != nil {
		return projectDetail{}, err
	}
	if _, err := dockerutil.GetDockerVersion(); err != nil {
		return projectDetail{}, err
	}
	defer dockerutil.CacheContainers()()
	desc, err := app.Describe(false)
	if err != nil {
		return projectDetail{}, err
	}
	d := projectDetail{Describe: desc, Addons: []addonInfo{}, Snapshots: []snapshotInfo{}}
	for _, a := range ddevapp.GetInstalledAddons(app) {
		d.Addons = append(d.Addons, addonInfo{Name: a.Name, Version: a.Version, Repository: a.Repository})
	}
	// Snapshots are a hint here, so ones that cannot be read are left out.
	targets, _ := app.ListSnapshotRestoreTargets()
	for _, t := range targets {
		d.Snapshots = append(d.Snapshots, snapshotInfo{Name: t.Name, Label: cmp.Or(t.Label, t.Name), Path: t.Path, Created: t.Created.Format("2006-01-02 15:04"), Size: t.Size})
	}
	// `ddev xdebug status` would start a stopped project, so the web
	// container is asked directly.
	if s, _ := app.SiteStatus(); s == ddevapp.SiteRunning {
		d.Xdebug = "off"
		if out, _, err := app.Exec(&ddevapp.ExecOpts{Cmd: "xdebug status"}); err == nil && strings.Contains(out, "xdebug enabled") {
			d.Xdebug = "on"
		}
	}
	return d, nil
}
