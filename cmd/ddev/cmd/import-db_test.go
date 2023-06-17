package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/testcommon"
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

	origDir, _ := os.Getwd()
	site := TestSites[0]
	err := os.Chdir(site.Dir)
	require.NoError(t, err)
	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)
	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		// Make sure all databases are back to default empty
		err = app.Stop(true, false)
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
	})

	err = app.Start()
	require.NoError(t, err)

	if app.IsMutagenEnabled() {
		_, _, longStatus, _ := app.MutagenStatus()
		t.Logf("mutagen status before show tables=%s", longStatus)
	}

	// Make sure we start with nothing in db
	out, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -N -e 'SHOW TABLES;'",
	})
	assert.NoError(err, "mysql exec output=%s", out)
	require.Equal(t, "", out)

	if app.IsMutagenEnabled() {
		_, _, longStatus, _ := app.MutagenStatus()
		t.Logf("mutagen status before opening file=%s", longStatus)
	}

	// Set up to read from the sql import file
	inputFile := filepath.Join(origDir, "testdata", t.Name(), "users.sql")
	f, err := os.Open(inputFile)
	require.NoError(t, err)
	// nolint: errcheck
	defer f.Close()

	if app.IsMutagenEnabled() {
		_, _, longStatus, _ := app.MutagenStatus()
		t.Logf("mutagen status before import-db=%s", longStatus)
	}
	// Run the import-db command with stdin coming from users.sql
	command := exec.Command(DdevBin, "import-db")
	command.Stdin = f

	importDBOutput, err := command.CombinedOutput()
	require.NoError(t, err, "failed import-db from stdin: %s (%v)", string(importDBOutput), err)

	assert.Contains(string(importDBOutput), "Successfully imported database")

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)
	assert.Equal("Tables_in_db\nusers\n", out)

	// Test with named project (outside project directory)
	// Test with named project (outside project directory)
	tmpDir := testcommon.CreateTmpDir(t.Name())
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
