package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/ddevapp"
	exec2 "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Cmd:     fmt.Sprintf(`%s -e 'SHOW TABLES;'`, app.GetDBClientCommand()),
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

	// Export with legacy flags, --compression, and type inferred from the extension
	cTypes := []struct {
		cType string
		ext   string
		flags []string
	}{
		{"gzip", "gz", []string{"--gzip"}},
		{"bzip2", "bz2", []string{"--bzip2"}},
		{"xz", "xz", []string{"--xz"}},
		{"zstd", "zst", []string{"--compression=zstd"}},
		{"zstd", "zst", nil},
		{"xz", "xz", nil},
	}
	outDir := filepath.Join(site.Dir, "tmp")
	err = os.MkdirAll(outDir, 0755)
	assert.NoError(err)
	err = os.Chdir(app.AppRoot)
	require.NoError(t, err)
	for _, c := range cTypes {
		cType, ext := c.cType, c.ext
		outputFile := filepath.Join(outDir, "db.sql."+ext)
		_ = os.RemoveAll(outputFile)
		out, err := exec2.RunHostCommand(DdevBin, append([]string{"export-db", "-f=" + outputFile}, c.flags...)...)
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
		case "zstd":
			err = archive.UnZstd(outputFile, outDir)
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
		Cmd:     fmt.Sprintf(`%s nondefault -e 'SHOW TABLES;'`, app.GetDBClientCommand()),
	})
	assert.NoError(err)

	outputFile = filepath.Join(tmpDir, "nondefault_output.sql")
	command = exec.Command(DdevBin, "export-db", site.Name, "-d=nondefault", "-f="+outputFile, "--gzip=false")
	byteout, err = command.CombinedOutput()
	assert.NoError(err, "export-db failure output=%s", string(byteout))
	assert.Contains(string(byteout), fmt.Sprintf("Wrote database dump from project '%s' database '%s' to file %s in plain text format", site.Name, "nondefault", outputFile))
	assert.FileExists(outputFile)
	assert.True(fileutil.FgrepStringInFile(outputFile, "INSERT INTO `nondefault_table` VALUES"))
	assert.True(fileutil.FgrepStringInFile(outputFile, "(0,'13751eca-19cf-41c2-90d4-9363f3a07c45','en'),"))

	_, _, err = app.Exec(&ddevapp.ExecOpts{
		Service: "db",
		Cmd:     fmt.Sprintf(`%s nondefault -e 'SELECT * FROM nondefault_table;'`, app.GetDBClientCommand()),
	})
	assert.NoError(err)
}

// TestResolveExportCompression checks how --compression, the legacy flags,
// and the --file extension combine, without needing a running project.
func TestResolveExportCompression(t *testing.T) {
	tests := []struct {
		args    []string
		file    string
		want    string
		wantErr bool
	}{
		{nil, "", "gzip", false},
		{nil, "/tmp/db.sql.gz", "gzip", false},
		{nil, "/tmp/db.SQL.ZST", "zstd", false},
		{nil, "/tmp/db.sql.xz", "xz", false},
		{nil, "/tmp/db.sql.bz2", "bzip2", false},
		{nil, "/tmp/db.sql", "none", false},
		{nil, "/tmp/db.mysql", "none", false},
		{nil, "/tmp/db.dump", "gzip", false},
		{[]string{"--compression=zstd"}, "", "zstd", false},
		{[]string{"--compression=none"}, "/tmp/db.sql.gz", "none", false},
		{[]string{"--compression=lz4"}, "", "", true},
		{[]string{"--gzip=false"}, "", "none", false},
		{[]string{"--gzip=false"}, "/tmp/db.sql.gz", "none", false},
		{[]string{"--xz"}, "/tmp/db.sql", "xz", false},
		{[]string{"--gzip=false", "--bzip2"}, "", "bzip2", false},
		{[]string{"--gzip", "--xz"}, "", "", true},
		{[]string{"--compression=gzip", "--gzip"}, "", "gzip", false},
		{[]string{"--compression=zstd", "--xz"}, "", "", true},
		{[]string{"--compression=zstd", "--gzip=false"}, "", "zstd", false},
		{[]string{"--compression=gzip", "--gzip=false"}, "", "", true},
	}
	for _, tc := range tests {
		cmd := NewExportDBCmd()
		require.NoError(t, cmd.ParseFlags(tc.args))
		got, err := resolveExportCompression(cmd.Flags(), tc.file)
		if tc.wantErr {
			require.Error(t, err, "args=%v file=%s", tc.args, tc.file)
			continue
		}
		require.NoError(t, err, "args=%v file=%s", tc.args, tc.file)
		require.Equal(t, tc.want, got, "args=%v file=%s", tc.args, tc.file)
	}
}
