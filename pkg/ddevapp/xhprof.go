package ddevapp

import (
	"github.com/ddev/ddev/pkg/config/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"strings"
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
		dbCreationCommand = `mysql -e "CREATE DATABASE IF NOT EXISTS xhgui; GRANT ALL ON xhgui.* to 'db'@'%'; GRANT ALL ON xhgui.* TO 'db'@'%';"`
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

// GetXHGuiURL returns teh URL for xhgui
func (app *DdevApp) GetXHGuiURL() string {
	baseURL := app.GetPrimaryURL()
	return baseURL + ":" + app.GetXHGuiPort()
}

// GetXHGuiPort returns the port where we're serving xhgui
// TODO: Make port configurable globally, etc.
func (app *DdevApp) GetXHGuiPort() string {
	// Normal case is https, port 8142
	if !app.CanUseHTTPOnly() {
		return "8142"
	}
	return "8143"
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
