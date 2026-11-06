package ddevapp

import (
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// resolvePortsLikeStart sets the router ports the way start() does, with
// substitutes standing in for ports that are busy.
func resolvePortsLikeStart(app *DdevApp, substitutes map[string]string) {
	app.restoreConfiguredRouterPorts()
	configured, configuredWebX := app.routerPortValues()

	app.RouterHTTPPort = app.GetPrimaryRouterHTTPPort()
	app.RouterHTTPSPort = app.GetPrimaryRouterHTTPSPort()
	app.MailpitHTTPPort = app.GetMailpitHTTPPort()
	app.MailpitHTTPSPort = app.GetMailpitHTTPSPort()
	app.XHGuiHTTPPort = app.GetXHGuiHTTPPort()
	app.XHGuiHTTPSPort = app.GetXHGuiHTTPSPort()
	AssignRouterPortsToGenericWebserverPorts(app)
	for _, field := range app.routerPortFields() {
		if s, ok := substitutes[*field]; ok {
			*field = s
		}
	}
	SyncGenericWebserverPortsWithRouterPorts(app)

	resolved, resolvedWebX := app.routerPortValues()
	app.resolvedRouterPorts = &resolvedRouterPorts{
		configured:     configured,
		resolved:       resolved,
		configuredWebX: configuredWebX,
		resolvedWebX:   resolvedWebX,
	}
}

// readWrittenConfig returns config.yaml as a generic map.
func readWrittenConfig(t *testing.T, app *DdevApp) map[string]any {
	t.Helper()
	content, err := os.ReadFile(app.ConfigPath)
	require.NoError(t, err)
	config := map[string]any{}
	require.NoError(t, yaml.Unmarshal(content, &config))
	return config
}

func newRouterPortsTestApp(t *testing.T) *DdevApp {
	t.Helper()
	projectName := strings.ToLower(t.Name())
	t.Cleanup(func() {
		_ = globalconfig.RemoveProjectInfo(projectName)
	})
	app, err := NewApp(t.TempDir(), false)
	require.NoError(t, err)
	app.Name = projectName
	app.Type = nodeps.AppTypePHP
	return app
}

func setGlobalRouterPorts(t *testing.T, mailpitHTTP, mailpitHTTPS, xhguiHTTP, xhguiHTTPS string) {
	t.Helper()
	g := &globalconfig.DdevGlobalConfig
	orig := []string{g.RouterMailpitHTTPPort, g.RouterMailpitHTTPSPort, g.RouterXHGuiHTTPPort, g.RouterXHGuiHTTPSPort}
	t.Cleanup(func() {
		g.RouterMailpitHTTPPort, g.RouterMailpitHTTPSPort, g.RouterXHGuiHTTPPort, g.RouterXHGuiHTTPSPort = orig[0], orig[1], orig[2], orig[3]
	})
	g.RouterMailpitHTTPPort, g.RouterMailpitHTTPSPort, g.RouterXHGuiHTTPPort, g.RouterXHGuiHTTPSPort = mailpitHTTP, mailpitHTTPS, xhguiHTTP, xhguiHTTPS
}

func TestWriteConfigOmitsResolvedRouterPorts(t *testing.T) {
	setGlobalRouterPorts(t, "18025", "18026", "", "")
	app := newRouterPortsTestApp(t)

	// The test environment's global config may set its own router ports.
	routerHTTPPort := app.GetPrimaryRouterHTTPPort()
	resolvePortsLikeStart(app, map[string]string{routerHTTPPort: "33000"})
	require.Equal(t, "33000", app.RouterHTTPPort)
	require.Equal(t, "18025", app.MailpitHTTPPort)
	require.Equal(t, nodeps.DdevDefaultXHGuiHTTPPort, app.XHGuiHTTPPort)
	require.NoError(t, app.WriteConfig())

	config := readWrittenConfig(t, app)
	for name := range app.routerPortFields() {
		require.NotContains(t, config, name)
	}
	// Writing the config must not change the resolved ports the running project uses.
	require.Equal(t, "33000", app.RouterHTTPPort)

	// A second start resolves from the configured ports again, so a freed port is used.
	resolvePortsLikeStart(app, nil)
	require.Equal(t, routerHTTPPort, app.RouterHTTPPort)
}

func TestWriteConfigKeepsProjectRouterPorts(t *testing.T) {
	setGlobalRouterPorts(t, "18025", "", "", "")
	app := newRouterPortsTestApp(t)
	app.MailpitHTTPPort = "28025"
	app.XHGuiHTTPSPort = "28142"

	resolvePortsLikeStart(app, map[string]string{"28025": "33001"})
	require.Equal(t, "33001", app.MailpitHTTPPort)
	// A port set after Start() is the project's own setting.
	app.RouterHTTPSPort = "8443"
	require.NoError(t, app.WriteConfig())

	config := readWrittenConfig(t, app)
	require.Equal(t, "28025", config["mailpit_http_port"])
	require.Equal(t, "28142", config["xhgui_https_port"])
	require.Equal(t, "8443", config["router_https_port"])
	require.NotContains(t, config, "router_http_port")
	require.NotContains(t, config, "xhgui_http_port")
}

func TestWriteConfigOmitsSubstituteGenericWebserverPorts(t *testing.T) {
	app := newRouterPortsTestApp(t)
	app.Type = nodeps.AppTypeGeneric
	app.WebserverType = nodeps.WebserverGeneric
	app.WebExtraExposedPorts = []WebExposedPort{{Name: "node", WebContainerPort: 3000, HTTPPort: 80, HTTPSPort: 443}}

	resolvePortsLikeStart(app, map[string]string{"80": "33000", "443": "33001"})
	require.Equal(t, 33000, app.WebExtraExposedPorts[0].HTTPPort)
	require.NoError(t, app.WriteConfig())

	written, err := NewApp(app.AppRoot, false)
	require.NoError(t, err)
	require.Len(t, written.WebExtraExposedPorts, 1)
	require.Equal(t, 80, written.WebExtraExposedPorts[0].HTTPPort)
	require.Equal(t, 443, written.WebExtraExposedPorts[0].HTTPSPort)
	require.Equal(t, 33000, app.WebExtraExposedPorts[0].HTTPPort)
}
