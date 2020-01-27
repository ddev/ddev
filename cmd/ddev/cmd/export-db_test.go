package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCmdExportDB does an export-db
func TestCmdExportDB(t *testing.T) {
	assert := asrt.New(t)

	testDir, _ := os.Getwd()
	site := TestSites[0]
	cleanup := site.Chdir()
	defer cleanup()
	app, err := ddevapp.NewApp(site.Dir, false, "")
	assert.NoError(err)

	// Read in a database
	inputFileName := filepath.Join(testDir, "testdata", t.Name(), "users.sql")
	_ = os.MkdirAll("tmp", 0755)
	// Run the import-db command with stdin coming from users.sql
	command := exec.Command(DdevBin, "import-db", "-f="+inputFileName)
	importDBOutput, err := command.CombinedOutput()
	require.NoError(t, err, "failed import-db from %s: %v", inputFileName, importDBOutput)

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql -e 'SHOW TABLES;'",
	})
	assert.NoError(err)

	// Export the database and verify content of output
	outputFileName := filepath.Join("tmp", t.Name(), "output.sql")
	_ = os.RemoveAll(outputFileName)
	command = exec.Command(DdevBin, "export-db", "-f="+outputFileName, "--gzip=false")
	_, err = command.CombinedOutput()
	assert.NoError(err)
	assert.FileExists(outputFileName)
	assert.True(fileutil.FgrepStringInFile(outputFileName, "13751eca-19cf-41c2-90d4-9363f3a07c45"))

	// Test with named project (outside project directory)
	err = os.Chdir("/tmp")
	assert.NoError(err)
	_ = os.RemoveAll(outputFileName)
	command = exec.Command(DdevBin, "export-db", site.Name, "-f="+outputFileName, "--gzip=false")
	_, err = command.CombinedOutput()
	assert.NoError(err)
	assert.FileExists(outputFileName)
	assert.True(fileutil.FgrepStringInFile(outputFileName, "13751eca-19cf-41c2-90d4-9363f3a07c45"))
}
