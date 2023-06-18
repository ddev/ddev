package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
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

	// Read in a database
	inputFileName := filepath.Join(origDir, "testdata", t.Name(), "users.sql")
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
	outputFile := filepath.Join(site.Dir, "tmp", t.Name(), "output.sql")
	_ = os.RemoveAll(outputFile)
	command = exec.Command(DdevBin, "export-db", "-f="+outputFile, "--gzip=false")
	byteout, err := command.CombinedOutput()
	require.NoError(t, err, "byteout=%s", string(byteout))
	assert.FileExists(outputFile)
	assert.True(fileutil.FgrepStringInFile(outputFile, "13751eca-19cf-41c2-90d4-9363f3a07c45"))

	// Test with named project (outside project directory)
	tmpDir := testcommon.CreateTmpDir("TestCmdExportDB")
	err = os.Chdir(tmpDir)
	assert.NoError(err)

	err = app.MutagenSyncFlush()
	assert.NoError(err)
	err = os.RemoveAll(outputFile)
	assert.NoError(err)
	command = exec.Command(DdevBin, "export-db", site.Name, "-f="+outputFile, "--gzip=false")
	byteout, err = command.CombinedOutput()
	assert.NoError(err, "export-db failure output=%s", string(byteout))
	assert.FileExists(outputFile)
	assert.True(fileutil.FgrepStringInFile(outputFile, "13751eca-19cf-41c2-90d4-9363f3a07c45"))

	// Export with various types of compression
	cTypes := map[string]string{
		"gzip":  "gz",
		"bzip2": "bz2",
		"xz":    "xz",
	}
	outDir := filepath.Join(site.Dir, "tmp")
	err = os.MkdirAll(outDir, 0755)
	assert.NoError(err)
	err = os.Chdir(app.AppRoot)
	require.NoError(t, err)
	for cType, ext := range cTypes {
		outputFile := filepath.Join(outDir, "db.sql."+ext)
		out, err := exec2.RunHostCommand(DdevBin, "export-db", "-f="+outputFile, "--"+cType)
		require.NoError(t, err, "export-db output failure for %s, output=%s", outputFile, out)
		require.FileExists(t, outputFile)
		uncompressedFile := filepath.Join(outDir, "db.sql")
		_ = os.RemoveAll(uncompressedFile)

		switch cType {
		case "gzip":
			err = archive.Ungzip(outputFile, outDir)
			assert.NoError(err)
		case "bzip2":
			err = archive.UnBzip2(outputFile, outDir)
			assert.NoError(err)
		case "xz":
			err = archive.UnXz(outputFile, outDir)
			assert.NoError(err)
		}
		assert.True(fileutil.FgrepStringInFile(uncompressedFile, "13751eca-19cf-41c2-90d4-9363f3a07c45"))
	}
	// Work with a non-default database named "nondefault"
	// Read in a database
	inputFileName = filepath.Join(origDir, "testdata", t.Name(), "nondefault.sql")
	// Run the import-db command with stdin coming from users.sql
	command = exec.Command(DdevBin, "import-db", site.Name, "-d=nondefault", "-f="+inputFileName)
	importDBOutput, err = command.CombinedOutput()
	require.NoError(t, err, "failed import-db from %s: %v", inputFileName, string(importDBOutput))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql nondefault -e 'SHOW TABLES;'",
	})
	assert.NoError(err)

	outputFile = filepath.Join(tmpDir, "nondefault_output.sql")
	command = exec.Command(DdevBin, "export-db", site.Name, "-d=nondefault", "-f="+outputFile, "--gzip=false")
	byteout, err = command.CombinedOutput()
	assert.NoError(err, "export-db failure output=%s", string(byteout))
	assert.Contains(string(byteout), fmt.Sprintf("Wrote database dump from project '%s' database '%s' to file %s in plain text format", site.Name, "nondefault", outputFile))
	assert.FileExists(outputFile)
	assert.True(fileutil.FgrepStringInFile(outputFile, "INSERT INTO `nondefault_table` VALUES (0,'13751eca-19cf-41c2-90d4-9363f3a07c45','en'),"))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     "mysql nondefault -e 'SELECT * FROM nondefault_table;'",
	})
	assert.NoError(err)
}
