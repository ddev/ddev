package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
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

	app, err := ddevapp.NewApp(site.Dir, false)
	assert.NoError(err)

	defer func() {
		// Make sure all databases are back to default empty
		_ = app.Stop(true, false)
		_ = app.Start()
		cleanup()
	}()

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
	_ = os.MkdirAll(filepath.Join("tmp", t.Name()), 0755)
	outputFileName := filepath.Join(site.Dir, "tmp", t.Name(), "output.sql")
	_ = os.RemoveAll(outputFileName)
	command = exec.Command(DdevBin, "export-db", "-f="+outputFileName, "--gzip=false")
	byteout, err := command.CombinedOutput()
	require.NoError(t, err, "byteout=%s", string(byteout))
	assert.FileExists(outputFileName)
	assert.True(fileutil.FgrepStringInFile(outputFileName, "13751eca-19cf-41c2-90d4-9363f3a07c45"))

	// Test with named project (outside project directory)
	tmpDir := testcommon.CreateTmpDir("TestCmdExportDB")
	err = os.Chdir(tmpDir)
	assert.NoError(err)

	err = os.RemoveAll(outputFileName)
	assert.NoError(err)
	command = exec.Command(DdevBin, "export-db", site.Name, "-f="+outputFileName, "--gzip=false")
	byteout, err = command.CombinedOutput()
	assert.NoError(err, "export-db failure output=%s", string(byteout))
	assert.FileExists(outputFileName)
	assert.True(fileutil.FgrepStringInFile(outputFileName, "13751eca-19cf-41c2-90d4-9363f3a07c45"))

	// Work with a non-default database named "nondefault"
	// Read in a database
	inputFileName = filepath.Join(testDir, "testdata", t.Name(), "nondefault.sql")
	// Run the import-db command with stdin coming from users.sql
	command = exec.Command(DdevBin, "import-db", site.Name, "-d=nondefault", "-f="+inputFileName)
	importDBOutput, err = command.CombinedOutput()
	require.NoError(t, err, "failed import-db from %s: %v", inputFileName, string(importDBOutput))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql nondefault -e 'SHOW TABLES;'",
	})
	assert.NoError(err)

	outputFileName = filepath.Join(tmpDir, "nondefault_output.sql")
	command = exec.Command(DdevBin, "export-db", site.Name, "-d=nondefault", "-f="+outputFileName, "--gzip=false")
	byteout, err = command.CombinedOutput()
	assert.NoError(err, "export-db failure output=%s", string(byteout))
	assert.Contains(string(byteout), fmt.Sprintf("Wrote database dump from %s database 'nondefault' to file %s in plain text format", site.Name, outputFileName))
	assert.FileExists(outputFileName)
	assert.True(fileutil.FgrepStringInFile(outputFileName, "INSERT INTO `nondefault_table` VALUES (0,'13751eca-19cf-41c2-90d4-9363f3a07c45','en'),"))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql nondefault -e 'SELECT * FROM nondefault_table;'",
	})
	assert.NoError(err)
}
