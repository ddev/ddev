package ddevapp

import (
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
			psql -q -c "SELECT 'CREATE DATABASE xhgui' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'xhgui')\gexec
			GRANT ALL PRIVILEGES ON DATABASE xhgui TO db;
			"
			`
	case nodeps.MySQL:
		fallthrough
	case nodeps.MariaDB:
		dbCreationCommand = `mysql -e "CREATE DATABASE IF NOT EXISTS xhgui; GRANT ALL ON xhgui.* to 'db'@'%'; GRANT ALL ON xhgui.* TO 'db'@'%';"`
	}

	_, _, err := app.Exec(&ExecOpts{
		Service: "db",
		Cmd:     dbCreationCommand,
	})
	if err != nil {
		return err
	}
	_, _, err = app.Exec(&ExecOpts{
		Cmd: `enable_xhprof`,
	})
	return err
}
