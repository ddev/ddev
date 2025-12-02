package ddevapp

import (
	"fmt"
	"strings"

	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
)

// XHGuiSetup does prerequisite work to make XHGui work
// - Creates the xhgui database if it does not exist
// - Enable xhprof
func XHGuiSetup(app *DdevApp) error {
	var dbCreationCommand string
	switch app.Database.Type {
	case nodeps.Postgres:
		dbCreationCommand = `
	set -eo -o pipefail; echo "SELECT 'CREATE DATABASE xhgui' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'xhgui')\gexec
	GRANT ALL PRIVILEGES ON DATABASE xhgui TO db;" | psql -q -d postgres
`
	case nodeps.MySQL, nodeps.MariaDB:
		dbCreationCommand = fmt.Sprintf(`%s -e "CREATE DATABASE IF NOT EXISTS xhgui; GRANT ALL ON xhgui.* TO 'db'@'%%';"`, app.GetDBClientCommand())
	}

	_, _, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     dbCreationCommand,
	})
	if err != nil {
		return err
	}

	if err = XHProfEnable(app); err != nil {
		return err
	}

	if !IsXHGuiContainerRunning(app) {
		err = app.StartOptionalProfiles([]string{"xhgui"})
		if err != nil {
			return err
		}
	}

	// Sync after enabling xhprof to ensure code files are available
	if err = app.MutagenSyncFlush(); err != nil {
		return err
	}

	return nil
}

// XHGuiStatus returns whether the `xhgui` container is running (and xhprof enabled)
func XHGuiStatus(app *DdevApp) (status bool) {
	if app.GetXHProfMode() != types.XHProfModeXHGui {
		return false
	}

	if xhprofStatus, _ := XHProfStatus(app); !xhprofStatus {
		return false
	}

	return IsXHGuiContainerRunning(app)
}

func IsXHGuiContainerRunning(app *DdevApp) bool {
	containerName := GetContainerName(app, "xhgui")
	container, err := dockerutil.FindContainerByName(containerName)
	if err == nil && container != nil {
		return true
	}
	return false
}

// GetXHGuiURL returns the URL for xhgui
func (app *DdevApp) GetXHGuiURL() string {
	var xhguiURL string

	if !IsRouterDisabled(app) {
		var desc, _ = app.Describe(true)

		if _, ok := desc["xhgui_url"]; ok {
			xhguiURL = desc["xhgui_url"].(string)
		}
		if _, ok := desc["xhgui_https_url"]; ok && !app.CanUseHTTPOnly() {
			xhguiURL = desc["xhgui_https_url"].(string)
		}
	} else {
		// If router is not enabled, use docker IP with regular port
		ip, _ := dockerutil.GetDockerIP()
		xhguiURL = fmt.Sprintf("http://%s:%s", ip, app.HostXHGuiPort)
	}

	return xhguiURL
}

// GetXHGuiPort returns the router port where we're serving xhgui
func (app *DdevApp) GetXHGuiPort() string {
	// Normal case is https, port 8142
	if !app.CanUseHTTPOnly() {
		return app.GetXHGuiHTTPSPort()
	}
	return app.GetXHGuiHTTPPort()
}

// XHProfEnable enables xhprof extension and starts gathering info
func XHProfEnable(app *DdevApp) error {
	_, _, err := app.Exec(&ExecOpts{
		Cmd: `enable_xhprof`,
	})
	return err
}

// XHProfDisable disables xhprof extension and stops gathering info
func XHProfDisable(app *DdevApp) error {
	_, _, err := app.Exec(&ExecOpts{
		Cmd: `disable_xhprof`,
	})
	return err
}

// XHProfStatus returns whether xhprof is enabled
func XHProfStatus(app *DdevApp) (status bool, err error) {
	out, _, err := app.Exec(&ExecOpts{
		Cmd: `php -r 'echo extension_loaded("xhprof");'`,
		Env: []string{"XDEBUG_MODE=off"},
	})
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(out, "1") {
		return true, nil
	}
	return false, nil
}

// GetXHProfMode returns xhprof mode config respecting defaults.
func (app *DdevApp) GetXHProfMode() types.XHProfMode {
	switch app.XHProfMode {
	case types.XHProfModeEmpty, types.XHProfModeGlobal:
		return globalconfig.DdevGlobalConfig.GetXHProfMode()
	default:
		return app.XHProfMode
	}
}

// GetXHGuiHTTPPort returns app's xhgui router http port
// If HTTP_EXPOSE has a mapping to port 8143 in the container, use that
// If not, use the global or project XHGuiHTTPPort
func (app *DdevApp) GetXHGuiHTTPPort() string {
	if httpExpose := app.GetXHGuiEnvVar("HTTP_EXPOSE"); httpExpose != "" {
		httpPort := app.TargetPortFromExposeVariable(httpExpose, "80")
		if httpPort != "" {
			return httpPort
		}
	}

	port := globalconfig.DdevGlobalConfig.RouterXHGuiHTTPPort
	if port == "" {
		port = nodeps.DdevDefaultXHGuiHTTPPort
	}
	if app.XHGuiHTTPPort != "" {
		port = app.XHGuiHTTPPort
	}
	return port
}

// GetXHGuiHTTPSPort returns app's xhgui router https port
// If HTTPS_EXPOSE has a mapping to port 8142 in the container, use that
// If not, use the global or project XHGuiHTTPSPort
func (app *DdevApp) GetXHGuiHTTPSPort() string {
	if httpsExpose := app.GetXHGuiEnvVar("HTTPS_EXPOSE"); httpsExpose != "" {
		httpsPort := app.TargetPortFromExposeVariable(httpsExpose, "80")
		if httpsPort != "" {
			return httpsPort
		}
	}

	port := globalconfig.DdevGlobalConfig.RouterXHGuiHTTPSPort
	if port == "" {
		port = nodeps.DdevDefaultXHGuiHTTPSPort
	}

	if app.XHGuiHTTPSPort != "" {
		port = app.XHGuiHTTPSPort
	}
	return port
}

// GetXHGuiEnvVar gets an environment variable from the xhgui service
// It returns empty string if there is no var or the ComposeYaml
// is just not set.
func (app *DdevApp) GetXHGuiEnvVar(name string) string {
	if app.ComposeYaml != nil && app.ComposeYaml.Services != nil {
		if service, ok := app.ComposeYaml.Services["xhgui"]; ok {
			if service.Environment != nil {
				if v, ok := service.Environment[name]; ok && v != nil {
					return *v
				}
			}
		}
	}
	return ""
}
