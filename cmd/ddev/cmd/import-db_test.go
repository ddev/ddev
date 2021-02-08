package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/testcommon"
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
	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)
	defer func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		_ = app.Start()
		cleanup()
	}()

	// Make sure we start with nothing in db
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -N -e 'SHOW TABLES;'",
	})
	assert.NoError(err, "mysql exec output=%s", out)
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
	require.NoError(t, err, "failed import-db from stdin: %v", importDBOutput)

	assert.Contains(string(importDBOutput), "Successfully imported database")

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)
	assert.Equal("Tables_in_db\nusers\n", out)

	// Test with named project (outside project directory)
	// Test with named project (outside project directory)
	tmpDir := testcommon.CreateTmpDir("TestCmdExportDB")
	err = os.Chdir(tmpDir)
	assert.NoError(err)

	// Run the import-db command with stdin coming from users.sql
	byteout, err := exec.Command(DdevBin, "import-db", app.Name, "--target-db=sparedb", "-f="+inputFile).CombinedOutput()
	assert.NoError(err, "failed import-db: %v (%s)", err, string(byteout))
	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     `echo "SELECT COUNT(*) FROM users;" | mysql -N sparedb`,
	})
	assert.NoError(err)
	assert.Equal("2\n", out)

}
