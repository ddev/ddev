package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
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

	importCmd := fmt.Sprintf("gzip -dc %s | %s import-db", filepath.Join(testDir, "testdata", t.Name(), "users.sql.gz"), DdevBin)
	out, err = exec.RunCommand("sh", []string{"-c", importCmd})
	assert.NoError(err)
	assert.Contains(out, "Successfully imported database")

	out, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)
	assert.Equal("Tables_in_db\nusers\n", out)
}
