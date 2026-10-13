package tui

import (
	"encoding/json"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// describeOutput is `ddev describe -j` for a running project, trimmed.
const describeOutput = `{"level":"info","msg":"table","raw":{
 "approot": "/home/user/apps/site", "database_type": "mysql", "database_version": "8.0",
 "dbinfo": {"host": "db", "dbPort": "3306", "published_port": 27685},
 "docroot": "", "mailpit_https_url": "https://site.ddev.site:8026", "mailpit_url": "http://site.ddev.site:8025",
 "mutagen_enabled": false, "name": "site", "nodejs_version": "24", "performance_mode": "none",
 "php_version": "8.5", "primary_url": "https://site.ddev.site", "router": "traefik", "router_disabled": false,
 "status": "running", "type": "php", "urls": ["https://site.ddev.site", "https://127.0.0.1:26873"],
 "webserver_type": "nginx-fpm", "xhgui_https_url": "https://site.ddev.site:8142", "xhgui_status": "disabled",
 "xhgui_url": "http://site.ddev.site:8143", "xhprof_mode": "xhgui",
 "services": {
  "xhgui": {"describe-url-port": "Launch: ddev xhgui", "describe-info": "", "exposed_ports": "80", "host_ports_mapping": [], "http_url": "http://site.ddev.site:8143", "https_url": "https://site.ddev.site:8142", "status": "stopped"},
  "web": {"host_http_url": "http://127.0.0.1:29727", "host_https_url": "https://127.0.0.1:26873", "exposed_ports": "80,443,8025",
   "host_ports_mapping": [{"exposed_port": "80", "host_port": "29727"}, {"exposed_port": "443", "host_port": "26873"}, {"exposed_port": "8025", "host_port": "23116"}],
   "http_url": "http://site.ddev.site", "https_url": "https://site.ddev.site", "status": "running"},
  "db": {"full_name": "ddev-site-db", "exposed_ports": "3306,33060", "host_ports_mapping": [{"exposed_port": "3306", "host_port": "27685"}], "status": "running"},
  "redis": {"describe-info": "redis:7", "exposed_ports": "6379", "status": "running"}
 }},"time":"2026-10-13T12:00:00+03:00"}
`

func parseDescribe(t *testing.T, edit func(raw map[string]any)) ProjectDetail {
	t.Helper()
	var line struct {
		Raw map[string]any `json:"raw"`
	}
	require.NoError(t, json.Unmarshal([]byte(describeOutput), &line))
	if edit != nil {
		edit(line.Raw)
	}
	b, err := json.Marshal(line.Raw)
	require.NoError(t, err)
	var d describeJSON
	require.NoError(t, json.Unmarshal(b, &d))
	return detailFromDescribe(d)
}

func TestDetailFromDescribe(t *testing.T) {
	d := parseDescribe(t, nil)
	require.Equal(t, "site", d.Name)
	require.Equal(t, ddevapp.SiteRunning, d.Status)
	require.Equal(t, "8.5", d.PHPVersion)
	require.Equal(t, "mysql", d.DatabaseType)
	require.Equal(t, "127.0.0.1:27685", d.DBHostPort)
	require.Equal(t, "https://site.ddev.site:8026", d.MailpitURL)
	require.Empty(t, d.XHGuiURL, "xhgui is disabled")
	require.Equal(t, []string{"https://site.ddev.site", "https://127.0.0.1:26873"}, d.URLs)

	var names []string
	for _, s := range d.Services {
		names = append(names, s.Name)
	}
	require.Equal(t, []string{"web", "db", "redis", "xhgui"}, names, "web, db, then by name")
	require.Equal(t, "https://site.ddev.site", d.Services[0].URL)
	require.Equal(t, []PortInfo{{"80", "29727"}, {"443", "26873"}, {"8025", "23116"}}, d.Services[0].Ports)
	require.Equal(t, []PortInfo{{"3306", "27685"}, {"33060", ""}}, d.Services[1].Ports, "33060 is not published")
	require.Equal(t, []PortInfo{{"6379", ""}}, d.Services[2].Ports)
	require.Equal(t, "127.0.0.1", d.DockerHost)
	require.Equal(t, "db:3306", d.DBInternal)
	require.Equal(t, "ddev-site-db:3306", d.DBShared, "other projects reach the database by its container name")
	require.Equal(t, "redis:7", d.Services[2].Info)
	require.Equal(t, "Launch: ddev xhgui", d.Services[3].Info)
}

func TestDetailFromDescribeVariants(t *testing.T) {
	stopped := parseDescribe(t, func(raw map[string]any) {
		raw["status"] = ddevapp.SiteStopped
		raw["dbinfo"] = map[string]any{"published_port": -1}
	})
	require.Empty(t, stopped.DBHostPort, "a stopped project has no published port")

	require.False(t, parseDescribe(t, nil).DBOmitted)
	noDB := parseDescribe(t, func(raw map[string]any) { delete(raw, "dbinfo") })
	require.True(t, noDB.DBOmitted, "describe leaves dbinfo out when omit_containers has db")
	require.Empty(t, noDB.DBHostPort)

	remote := parseDescribe(t, func(raw map[string]any) {
		web := raw["services"].(map[string]any)["web"].(map[string]any)
		web["host_https_url"] = "https://192.168.5.2:26873"
	})
	require.Equal(t, "192.168.5.2:27685", remote.DBHostPort, "the published port is on the Docker host")

	httpOnly := parseDescribe(t, func(raw map[string]any) {
		raw["primary_url"] = "http://site.ddev.site"
		raw["xhgui_status"] = "enabled"
	})
	require.Equal(t, "http://site.ddev.site:8025", httpOnly.MailpitURL)
	require.Equal(t, "http://site.ddev.site:8143", httpOnly.XHGuiURL)
	require.Equal(t, "http://site.ddev.site", httpOnly.Services[0].URL)

	xhgui := parseDescribe(t, func(raw map[string]any) { raw["xhgui_status"] = "enabled" })
	require.Equal(t, "https://site.ddev.site:8142", xhgui.XHGuiURL)

	noRouter := parseDescribe(t, func(raw map[string]any) { raw["router_disabled"] = true })
	require.Empty(t, noRouter.MailpitURL, "Mailpit is reached through the router")
	require.Equal(t, "https://127.0.0.1:26873", noRouter.Services[0].URL)
}

func TestHealthProblem(t *testing.T) {
	require.Equal(t, "traefik: config error", health{"unhealthy", "traefik:\n  config error"}.problem(), "the log is flattened to one line")
	require.Empty(t, health{"healthy", "sleeping..."}.problem(), "a healthy container's log is routine")
}

func TestLogBuffer(t *testing.T) {
	b := &logBuffer{}
	require.Empty(t, b.take())
	_, _ = b.Write([]byte("first warning\n\x1b[33mProject 'gone' was removed\x1b[0m\n"))
	require.Equal(t, "Project 'gone' was removed", b.take(), "the last line, without colors")
	require.Empty(t, b.take(), "taking empties the buffer")
}

func TestCaptureLogsDropsDebug(t *testing.T) {
	level := output.UserOut.GetLevel()
	t.Cleanup(func() { output.UserOut.SetLevel(level) })
	output.UserOut.SetLevel(logrus.DebugLevel)
	restore := captureLogs()
	output.UserOut.Warn("a warning")
	output.UserOut.Debug("a debug line")
	got := logs.take()
	restore()
	require.Contains(t, got, "a warning", "debug output would replace the status line on every load")
	require.Equal(t, logrus.DebugLevel, output.UserOut.GetLevel(), "the level comes back after the TUI")
}

func TestDdevCommandDisablesTUI(t *testing.T) {
	useFakeDdev(t)
	c, err := ddevCommand("/some/dir", "start")
	require.NoError(t, err)
	require.Equal(t, "/some/dir", c.Dir)
	require.Contains(t, c.Env, "DDEV_NO_TUI=true", "a ddev subcommand must not open a nested TUI")

	orig := baseEnv
	t.Cleanup(func() { baseEnv = orig })
	baseEnv = []string{"PATH=/bin"}
	t.Setenv("DDEV_SITENAME", "described")
	c, err = ddevCommand("", "start")
	require.NoError(t, err)
	require.Equal(t, []string{"PATH=/bin", "DDEV_NO_TUI=true"}, c.Env, "a described project's variables are not passed on")
}
