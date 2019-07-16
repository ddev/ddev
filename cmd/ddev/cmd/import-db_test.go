package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCmdImportDB does an import from stdin; many other cases are covered in
// TestDdevImportDB. This assumes an empty starting database.
func TestCmdImportDB(t *testing.T) {
	assert := asrt.New(t)

	testDir, _ := os.Getwd()
	site := TestSites[0]
	cleanup := site.Chdir()
	defer cleanup()
	app, err := ddevapp.NewApp(site.Dir, false, "")
	assert.NoError(err)

	// Make sure we start with nothing in db
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)
	require.Equal(t, "", out)

	// Set up to read from the sql import file
	inputFile := filepath.Join(testDir, "testdata", t.Name(), "users.sql")
	f, err := os.Open(inputFile)
	require.NoError(t, err)
	// nolint: errcheck
	defer f.Close()

	// Run the import-db command with stdin coming from users.sql
	command := exec.Command(DdevBin, "import-db")
	command.Stdin = f

	importDBOutput, err := command.CombinedOutput()
	require.NoError(t, err)

	assert.Contains(string(importDBOutput), "Successfully imported database")

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)
	assert.Equal("Tables_in_db\nusers\n", out)
}
