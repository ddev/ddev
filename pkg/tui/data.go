package tui

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/sirupsen/logrus"
)

// ProjectInfo holds what the project list shows for a DDEV project.
type ProjectInfo struct {
	Name   string
	Status string
	Type   string
	// URL is the primary URL, set only while the project is running.
	URL     string
	AppRoot string
	// Detail is the part of the detail view the configuration gives, shown
	// until Describe loads the rest.
	Detail *ProjectDetail
}

// ProjectDetail holds what the detail view shows, read from the project's
// description and installed add-ons.
type ProjectDetail struct {
	Name            string
	Status          string
	Type            string
	AppRoot         string
	PHPVersion      string
	WebserverType   string
	NodeJSVersion   string
	Docroot         string
	PerformanceMode string
	XHProfMode      string
	DatabaseType    string
	DatabaseVersion string
	// DBOmitted is set when omit_containers leaves the database out.
	DBOmitted bool
	// DockerHost is the address published ports listen on.
	DockerHost string
	// DBHostPort is empty unless the database port is published.
	DBHostPort string
	// DBInternal is the database's address from this project's containers,
	// and DBShared from other projects' containers.
	DBInternal    string
	DBShared      string
	PrimaryURL    string
	URLs          []string
	MailpitURL    string
	XHGuiURL      string
	Router        string
	MutagenStatus string
	Addons        []AddonInfo
	// Snapshots are what `ddev snapshot restore` offers, newest first, so
	// the latest is what --latest restores, even from another git worktree.
	Snapshots []ddevapp.SnapshotRestoreTarget
	Services  []ServiceInfo
	// FromConfig is set while only the configuration is known, without the
	// status, URLs, services and add-ons.
	FromConfig bool
}

// AddonInfo is an installed add-on.
type AddonInfo struct {
	Name       string
	Version    string
	Repository string
}

// ServiceInfo is one of a project's services.
type ServiceInfo struct {
	Name   string
	Status string
	URL    string
	Ports  []PortInfo
	// Info is the add-on's x-ddev describe-url-port and describe-info.
	Info string
}

// PortInfo is a port a service exposes, and the Docker host port it is
// published on, if any.
type PortInfo struct {
	Container string
	Host      string
}

// describeJSON is the part of a project's description the detail view reads.
type describeJSON struct {
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Type            string   `json:"type"`
	AppRoot         string   `json:"approot"`
	PHPVersion      string   `json:"php_version"`
	WebserverType   string   `json:"webserver_type"`
	NodeJSVersion   string   `json:"nodejs_version"`
	Docroot         string   `json:"docroot"`
	PerformanceMode string   `json:"performance_mode"`
	XHProfMode      string   `json:"xhprof_mode"`
	XHGuiStatus     string   `json:"xhgui_status"`
	XHGuiURL        string   `json:"xhgui_url"`
	XHGuiHTTPSURL   string   `json:"xhgui_https_url"`
	MailpitURL      string   `json:"mailpit_url"`
	MailpitHTTPSURL string   `json:"mailpit_https_url"`
	PrimaryURL      string   `json:"primary_url"`
	URLs            []string `json:"urls"`
	Router          string   `json:"router"`
	RouterDisabled  bool     `json:"router_disabled"`
	MutagenStatus   string   `json:"mutagen_status"`
	DatabaseType    string   `json:"database_type"`
	DatabaseVersion string   `json:"database_version"`
	DBInfo          *struct {
		Host          string `json:"host"`
		Port          string `json:"dbPort"`
		PublishedPort int    `json:"published_port"`
	} `json:"dbinfo"`
	Services map[string]struct {
		Status       string `json:"status"`
		HTTPURL      string `json:"http_url"`
		HTTPSURL     string `json:"https_url"`
		HostHTTPURL  string `json:"host_http_url"`
		HostHTTPSURL string `json:"host_https_url"`
		ExposedPorts string `json:"exposed_ports"`
		HostPorts    []struct {
			Exposed string `json:"exposed_port"`
			Host    string `json:"host_port"`
		} `json:"host_ports_mapping"`
		DescribeURLPort string `json:"describe-url-port"`
		DescribeInfo    string `json:"describe-info"`
	} `json:"services"`
}

// statusLoading is the status of a project or shared container until
// Docker is asked.
const statusLoading = "loading"

// containerDisabled is the status of a shared container that every project
// leaves out with omit_containers, or that the environment cannot run.
const containerDisabled = "disabled"

// ddevMu serializes the loaders, which run in parallel but share ddevapp's
// global state, such as the project list.
var ddevMu sync.Mutex

// health is a container's health status and its latest healthcheck output.
type health struct {
	status string
	log    string
}

// problem returns the healthcheck output when the container is not healthy.
// A healthy container's output is routine, so it is left out.
func (h health) problem() string {
	if h.status == "healthy" {
		return ""
	}
	return strings.Join(strings.Fields(h.log), " ")
}

// readProjects lists the registered projects from their configuration,
// without asking Docker, so the list shows before their status is known.
func readProjects() tea.Msg {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	_ = globalconfig.ReadProjectList()
	var projects []ProjectInfo
	for _, info := range globalconfig.GetGlobalProjectList() {
		// Projects that fail to load are left to loadProjects to report.
		if app, err := ddevapp.NewApp(info.AppRoot, true); err == nil {
			projects = append(projects, listedInfo(app))
		}
	}
	slices.SortFunc(projects, func(a, b ProjectInfo) int { return strings.Compare(a.Name, b.Name) })
	logs.take()
	return projectsListedMsg{projects: projects}
}

// loadProjects fetches the project list and the shared containers' health.
func loadProjects() tea.Msg {
	ddevMu.Lock()
	defer ddevMu.Unlock()
	// Projects that `ddev config` added since the last load are on disk only.
	if err := globalconfig.ReadProjectList(); err != nil {
		return projectsLoadedMsg{err: err}
	}
	// Without Docker every project would show as stopped.
	if _, err := dockerutil.GetDockerVersion(); err != nil {
		return projectsLoadedMsg{err: err, warning: logs.take()}
	}
	apps, err := ddevapp.GetProjects(false)
	if err != nil {
		return projectsLoadedMsg{err: err, warning: logs.take()}
	}
	msg := projectsLoadedMsg{projects: make([]ProjectInfo, 0, len(apps))}
	routerUsed, sshAgentUsed := len(apps) == 0, len(apps) == 0
	for _, app := range apps {
		msg.projects = append(msg.projects, projectInfo(app))
		routerUsed = routerUsed || !ddevapp.IsRouterDisabled(app)
		sshAgentUsed = sshAgentUsed || !slices.Contains(app.GetOmittedContainers(), globalconfig.DdevSSHAgentContainer)
	}
	msg.router, msg.sshAgent = health{status: containerDisabled}, health{status: containerDisabled}
	if routerUsed {
		msg.router.status, msg.router.log = ddevapp.GetRouterStatus()
	}
	if sshAgentUsed {
		msg.sshAgent.status, msg.sshAgent.log = ddevapp.GetSSHAuthStatus()
	}
	msg.warning = logs.take()
	return msg
}

// listedInfo is what the project list shows from the configuration alone.
func listedInfo(app *ddevapp.DdevApp) ProjectInfo {
	return ProjectInfo{Name: app.GetName(), Status: statusLoading, Type: app.GetType(), AppRoot: app.GetAppRoot(), Detail: configDetail(app)}
}

// projectInfo reads what the project list shows, the way `ddev list` does.
func projectInfo(app *ddevapp.DdevApp) ProjectInfo {
	info := listedInfo(app)
	info.Status, _ = app.SiteStatus()
	if info.Status == ddevapp.SiteRunning {
		info.URL = app.GetPrimaryURL()
	}
	return info
}

// configDetail is the part of a project's detail its configuration holds,
// read the way Describe reads it.
func configDetail(app *ddevapp.DdevApp) *ProjectDetail {
	d := &ProjectDetail{
		Name:            app.GetName(),
		Status:          statusLoading,
		Type:            app.GetType(),
		AppRoot:         app.GetAppRoot(),
		PHPVersion:      app.GetPhpVersion(),
		WebserverType:   app.GetWebserverType(),
		NodeJSVersion:   app.NodeJSVersion,
		Docroot:         app.GetDocroot(),
		PerformanceMode: app.GetPerformanceMode(),
		XHProfMode:      app.GetXHProfMode(),
		DatabaseType:    app.Database.Type,
		DatabaseVersion: app.Database.Version,
		DBOmitted:       app.IsDBOmitted(),
		FromConfig:      true,
	}
	d.DBInternal, d.DBShared = dbAddresses(d.Name, ddevapp.GetInternalPort(app, "db"))
	return d
}

// dbAddresses are the database's addresses from this project's containers and
// from other projects' containers. Both follow from the configuration, so they
// show before Describe returns and while the project is stopped.
func dbAddresses(project, port string) (internal, shared string) {
	return net.JoinHostPort("db", port), net.JoinHostPort("ddev-"+project+"-db", port)
}

// loadDetailCmd fetches a project's detail in the background.
func loadDetailCmd(appRoot string) tea.Cmd {
	return func() tea.Msg {
		ddevMu.Lock()
		defer ddevMu.Unlock()
		detail, err := loadDetail(appRoot)
		return projectDetailLoadedMsg{appRoot: appRoot, detail: detail, err: err, warning: logs.take()}
	}
}

// loadDetail reads a project's description and installed add-ons. The
// caller holds ddevMu.
func loadDetail(appRoot string) (ProjectDetail, error) {
	if _, err := os.Stat(appRoot); errors.Is(err, os.ErrNotExist) {
		return ProjectDetail{}, fmt.Errorf("project directory no longer exists: %s", appRoot)
	}
	app, err := ddevapp.NewApp(appRoot, true)
	if err != nil {
		return ProjectDetail{}, err
	}
	desc, err := app.Describe(false)
	if err != nil {
		return ProjectDetail{}, err
	}
	// The description is the map `ddev describe -j` prints, which
	// describeJSON reads with types.
	var d describeJSON
	b, err := json.Marshal(desc)
	if err == nil {
		err = json.Unmarshal(b, &d)
	}
	if err != nil {
		return ProjectDetail{}, fmt.Errorf("reading the description of %s: %w", app.Name, err)
	}
	detail := detailFromDescribe(d)
	for _, a := range ddevapp.GetInstalledAddons(app) {
		detail.Addons = append(detail.Addons, AddonInfo{Name: a.Name, Version: a.Version, Repository: a.Repository})
	}
	// The snapshots are only a hint here, so one that cannot be read is
	// left out rather than failing the detail.
	detail.Snapshots, _ = app.ListSnapshotRestoreTargets()
	return detail, nil
}

// xdebugCmd runs the web container's xdebug script with arg, which is
// status, on or off, in the project at appRoot. It goes through the Docker
// API, since `ddev xdebug` would start a stopped project.
func xdebugCmd(appRoot, arg string) tea.Cmd {
	return func() tea.Msg {
		ddevMu.Lock()
		defer ddevMu.Unlock()
		defer logs.take()
		msg := xdebugStatusMsg{appRoot: appRoot, toggled: arg != "status"}
		app, err := ddevapp.NewApp(appRoot, true)
		if err != nil {
			msg.err = err
			return msg
		}
		out, stderr, err := app.Exec(&ddevapp.ExecOpts{Cmd: "xdebug " + arg})
		if err != nil {
			msg.err = fmt.Errorf("xdebug %s: %s", arg, cmp.Or(lastLine(stderr), err.Error()))
			return msg
		}
		msg.enabled = arg == "on" || (arg == "status" && strings.Contains(out, "xdebug enabled"))
		return msg
	}
}

// latestAddonVersions maps each add-on's lowercased repository to its latest
// release, from the add-on registry ddev caches for a day. That needs no
// GitHub request per add-on, unlike `ddev add-on update`. It runs without
// ddevMu, so refreshing a stale cache cannot hold up the other loaders.
func latestAddonVersions() map[string]string {
	addons, err := ddevapp.ListAvailableAddonsFromRegistry()
	if err != nil {
		return nil
	}
	versions := map[string]string{}
	for _, a := range addons {
		if a.TagName.Value != "" {
			versions[strings.ToLower(a.User+"/"+a.Repo)] = a.TagName.Value
		}
	}
	return versions
}

// logBuffer holds what ddevapp logs while the TUI owns the terminal, which
// would otherwise print over the screen.
type logBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

var logs = &logBuffer{}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// take empties the buffer and returns the last line logged.
func (l *logBuffer) take() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.buf.String()
	l.buf.Reset()
	return lastLine(s)
}

// captureLogs sends ddevapp's logs to the buffer until restore is called.
// Debug output, as with DDEV_DEBUG, is dropped: its timestamped lines would
// replace the status line on every load.
func captureLogs() (restore func()) {
	out, errOut, level := output.UserOut.Out, output.UserErr.Out, output.UserOut.GetLevel()
	output.UserOut.SetOutput(logs)
	output.UserErr.SetOutput(logs)
	output.UserOut.SetLevel(min(level, logrus.InfoLevel))
	return func() {
		output.UserOut.SetOutput(out)
		output.UserErr.SetOutput(errOut)
		output.UserOut.SetLevel(level)
	}
}

// detailFromDescribe converts `ddev describe -j` output to a ProjectDetail.
func detailFromDescribe(d describeJSON) ProjectDetail {
	detail := ProjectDetail{
		Name:            d.Name,
		Status:          d.Status,
		Type:            d.Type,
		AppRoot:         d.AppRoot,
		PHPVersion:      d.PHPVersion,
		WebserverType:   d.WebserverType,
		NodeJSVersion:   d.NodeJSVersion,
		Docroot:         d.Docroot,
		PerformanceMode: d.PerformanceMode,
		XHProfMode:      d.XHProfMode,
		DatabaseType:    d.DatabaseType,
		DatabaseVersion: d.DatabaseVersion,
		DBOmitted:       d.DBInfo == nil,
		PrimaryURL:      d.PrimaryURL,
		URLs:            d.URLs,
		Router:          d.Router,
		MutagenStatus:   d.MutagenStatus,
	}
	// A project that can only use HTTP has an http:// primary URL, and its
	// HTTPS Mailpit and XHGui ports are not reachable.
	httpOnly := strings.HasPrefix(d.PrimaryURL, "http://")
	if !d.RouterDisabled {
		detail.MailpitURL = d.MailpitHTTPSURL
		if httpOnly || detail.MailpitURL == "" {
			detail.MailpitURL = d.MailpitURL
		}
		if d.XHGuiStatus == "enabled" {
			detail.XHGuiURL = d.XHGuiHTTPSURL
			if httpOnly || detail.XHGuiURL == "" {
				detail.XHGuiURL = d.XHGuiURL
			}
		}
	}
	// Published ports listen on the Docker host, which the web container's
	// direct URL already names.
	detail.DockerHost = "127.0.0.1"
	if web, ok := d.Services["web"]; ok {
		if u, err := url.Parse(cmp.Or(web.HostHTTPSURL, web.HostHTTPURL)); err == nil && u.Hostname() != "" {
			detail.DockerHost = u.Hostname()
		}
	}
	if d.DBInfo != nil {
		if d.DBInfo.PublishedPort > 0 {
			detail.DBHostPort = net.JoinHostPort(detail.DockerHost, strconv.Itoa(d.DBInfo.PublishedPort))
		}
		if d.DBInfo.Port != "" {
			detail.DBInternal, detail.DBShared = dbAddresses(d.Name, d.DBInfo.Port)
		}
	}
	for name, s := range d.Services {
		svc := ServiceInfo{
			Name:   name,
			Status: s.Status,
			URL:    cmp.Or(s.HTTPSURL, s.HTTPURL),
			Info:   strings.TrimSpace(strings.Join([]string{s.DescribeURLPort, s.DescribeInfo}, " ")),
		}
		published := map[string]string{}
		for _, p := range s.HostPorts {
			published[p.Exposed] = p.Host
		}
		for port := range strings.SplitSeq(s.ExposedPorts, ",") {
			if port = strings.TrimSpace(port); port != "" {
				svc.Ports = append(svc.Ports, PortInfo{Container: port, Host: published[port]})
			}
		}
		if httpOnly {
			svc.URL = cmp.Or(s.HTTPURL, s.HTTPSURL)
		}
		if d.RouterDisabled {
			svc.URL = cmp.Or(s.HostHTTPSURL, s.HostHTTPURL)
		}
		detail.Services = append(detail.Services, svc)
	}
	sortServices(detail.Services)
	return detail
}

// sortServices puts web first, db second, and the rest by name.
func sortServices(services []ServiceInfo) {
	order := func(name string) string {
		switch name {
		case "web":
			return "0"
		case "db":
			return "1"
		}
		return "2" + name
	}
	slices.SortFunc(services, func(a, b ServiceInfo) int { return strings.Compare(order(a.Name), order(b.Name)) })
}

// currentProjectRoot returns the app root of the project containing dir, or
// "" when dir is not inside any project.
func currentProjectRoot(projects []ProjectInfo, dir string) string {
	best := ""
	for _, p := range projects {
		if p.AppRoot == "" {
			continue
		}
		rel, err := filepath.Rel(p.AppRoot, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			continue
		}
		if len(p.AppRoot) > len(best) {
			best = p.AppRoot
		}
	}
	return best
}
