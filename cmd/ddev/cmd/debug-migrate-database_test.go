package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestDebugMigrateDatabase checks to see if we can migrate database
func TestDebugMigrateDatabase(t *testing.T) {
	origDir, _ := os.Getwd()

	site := TestSites[0]
	_ = os.Chdir(site.Dir)

	app, err := ddevapp.NewApp(site.Dir, false)
	require.NoError(t, err)

	// Remove existing
	err = app.Stop(true, false)
	require.NoError(t, err)

	// Use notorious mariadb:11.8 as source and notorious mysql:8.4 as target
	app.Database.Type = nodeps.MariaDB
	app.Database.Version = nodeps.MariaDB118

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "debug", "migrate-database", fmt.Sprintf("%s:%s", nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
		require.NoError(t, err, "failed to migrate database; out='%s'", out)

		require.Contains(t, out, fmt.Sprintf("Database was converted to %s:%s", nodeps.MariaDB, nodeps.MariaDBDefaultVersion))

		out, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     fmt.Sprintf(`%s -e 'DROP TABLE IF EXISTS %s;'`, app.GetDBClientCommand(), t.Name()),
		})
		require.NoError(t, err, "DROP table didn't work, out='%s', stderr='%s'", out, stderr)
		_ = os.Chdir(origDir)
	})

	err = app.Start()
	require.NoError(t, err)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`%s -N -e 'SELECT VERSION();'`, app.GetDBClientCommand()),
	})
	require.NoError(t, err)
	// It should have our default version
	require.True(t, strings.HasPrefix(out, nodeps.MariaDB118))

	// Import a database so we have something to work with
	err = app.ImportDB(filepath.Join(origDir, "testdata", t.Name(), "users.sql"), "", false, false, "")
	require.NoError(t, err)

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`%s -e 'CREATE TABLE IF NOT EXISTS example_table (name VARCHAR(255) NOT NULL); INSERT INTO example_table (name) VALUES ("%s");'`, app.GetDBClientCommand(), t.Name()),
	})
	require.NoError(t, err)

	// Try a migration
	out, err = exec.RunHostCommand(DdevBin, "utility", "migrate-database", fmt.Sprintf("%s:%s", nodeps.MySQL, nodeps.MySQL84))
	require.NoError(t, err, "failed to migrate database; out='%s'", out)
	require.Contains(t, out, fmt.Sprintf("Database was converted to %s:%s", nodeps.MySQL, nodeps.MySQL84))

	_, err = app.ReadConfig(true)
	require.NoError(t, err)

	// Make sure our inserted data is still there
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`%s -N -e 'SELECT name FROM example_table WHERE name = "%s";'`, app.GetDBClientCommand(), t.Name()),
	})
	require.NoError(t, err)
	require.Contains(t, out, t.Name())

	// Make sure we have the expected new version
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`%s -N  -e 'SELECT VERSION();'`, app.GetDBClientCommand()),
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, nodeps.MySQL84))
}
