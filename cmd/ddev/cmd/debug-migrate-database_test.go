package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/nodeps"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

// TestDebugMigrateDatabase checks to see if we can migrate database
// This does only a trivial change between two mariadb versions
func TestDebugMigrateDatabase(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	site := TestSites[0]
	_ = os.Chdir(site.Dir)

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	app.Database.Type = nodeps.MariaDB
	app.Database.Version = nodeps.MariaDB104

	t.Cleanup(func() {
		_ = os.Chdir(origDir)

		err = app.Stop(true, false)
		assert.NoError(err)
	})

	err = app.Start()
	require.NoError(t, err)

	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `mysql -N -e 'SELECT VERSION();'`,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(out, nodeps.MariaDB104))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS example_table (name VARCHAR(255) NOT NULL); INSERT INTO example_table (name) VALUES ("%s");'`, t.Name()),
	})
	require.NoError(t, err)

	// Try a migration
	out, err = exec.RunHostCommand(DdevBin, "debug", "migrate-database", "mariadb:10.8")
	require.NoError(t, err, "failed to migrate database; out='%s'", out)

	require.Contains(t, out, "database was converted to mariadb:10.8")

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
	require.True(t, strings.HasPrefix(out, nodeps.MariaDB108))
}
