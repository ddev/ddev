package cmd

import (
	"path/filepath"
	"testing"

	"os"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	asrt "github.com/stretchr/testify/assert"
)

// TestImportTilde tests passing paths to import-files that use ~ to represent home dir.
func TestImportTilde(t *testing.T) {
	assert := asrt.New(t)

	site := TestSites[0]

	homedir, err := os.UserHomeDir()
	assert.NoError(err)
	cwd, _ := os.Getwd()
	testFile := filepath.Join(homedir, "testfile.tar.gz")
	err = fileutil.CopyFile(filepath.Join(cwd, "testdata", "testfile.tar.gz"), testFile)
	assert.NoError(err)

	cleanup := site.Chdir()
	defer rmFile(testFile)

	// this ~ should be expanded by shell
	args := []string{"import-files", "--source", "~/testfile.tar.gz"}
	out, err := exec.RunCommand(DdevBin, args)
	if err != nil {
		t.Log("Error Output from ddev import-files:", out, site)
	}
	assert.NoError(err)
	assert.Contains(string(out), "Successfully imported files")

	// this ~ is not expanded by shell, ddev should convert it to a valid path
	args = []string{"import-files", "--source=~/testfile.tar.gz"}
	out, err = exec.RunCommand(DdevBin, args)
	if err != nil {
		t.Log("Error Output from ddev import-files:", out, site)
	}
	assert.NoError(err)
	assert.Contains(string(out), "Successfully imported files")

	cleanup()

	assert.NoError(nil)
}

// rmFile simply allows us to defer os.Remove while ignoring the error return.
func rmFile(fullPath string) {
	_ = os.Remove(fullPath)
}
