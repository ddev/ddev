package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDebugMigrateDatabase checks to see if we can migrate database
func TestDebugMigrateDatabase(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	_ = os.Chdir(site.Dir)

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	app.Database.Type = nodeps.MariaDB
	app.Database.Version = nodeps.MariaDBDefaultVersion

	t.Cleanup(func() {
		out, err := exec.RunHostCommand(DdevBin, "debug", "migrate-database", fmt.Sprintf("%s:%s", nodeps.MariaDB, nodeps.MariaDBDefaultVersion))
		assert.NoError(err, "failed to migrate database; out='%s'", out)

		assert.Contains(out, fmt.Sprintf("database was converted to %s:%s", nodeps.MariaDB, nodeps.MariaDBDefaultVersion))

		out, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "db",
			Cmd:     fmt.Sprintf(`mysql -e 'DROP TABLE IF EXISTS %s;'`, t.Name()),
		})
		assert.NoError(err, "DROP table didn't work, out='%s', stderr='%s'", out, stderr)
		_ = os.Chdir(origDir)
	})

	err = app.Start()
	require.NoError(t, err)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `mysql -N -e 'SELECT VERSION();'`,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, nodeps.MariaDB104))

	// Import a database so we have something to work with
	err = app.ImportDB(filepath.Join(origDir, "testdata", t.Name(), "users.sql"), "", false, false, "")
	require.NoError(t, err)

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS example_table (name VARCHAR(255) NOT NULL); INSERT INTO example_table (name) VALUES ("%s");'`, t.Name()),
	})
	require.NoError(t, err)

	// Try a migration
	out, err = exec.RunHostCommand(DdevBin, "debug", "migrate-database", fmt.Sprintf("%s:%s", nodeps.MySQL, nodeps.MySQL80))
	require.NoError(t, err, "failed to migrate database; out='%s'", out)
	require.Contains(t, out, fmt.Sprintf("database was converted to %s:%s", nodeps.MySQL, nodeps.MySQL80))

	// Make sure our inserted data is still there
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`mysql -N -e 'SELECT name FROM example_table WHERE name = "%s";'`, t.Name()),
	})
	require.NoError(t, err)
	require.Contains(t, out, t.Name())

	// Make sure we have the expected new version
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `mysql -N  -e 'SELECT VERSION();'`,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, nodeps.MySQL80))
}
