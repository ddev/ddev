package cmd

import (
	"log"
	"path/filepath"
	"testing"

	"os"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

// TestImportTilde tests passing paths to import-files that use ~ to represent home dir.
func TestImportTilde(t *testing.T) {
	assert := assert.New(t)

	for _, site := range DevTestSites {

		homedir, err := homedir.Dir()
		assert.NoError(err)
		cwd, _ := os.Getwd()
		testFile := filepath.Join(homedir, "testfile.tar.gz")
		err = fileutil.CopyFile(filepath.Join(cwd, "testdata", "testfile.tar.gz"), testFile)
		assert.NoError(err)

		cleanup := site.Chdir()
		defer os.Remove(testFile)

		// this ~ should be expanded by shell
		args := []string{"import-files", "--src", "~/testfile.tar.gz"}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Println("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		// this ~ is not expanded by shell, ddev should convert it to a valid path
		args = []string{"import-files", "--src=~/testfile.tar.gz"}
		out, err = exec.RunCommand(DdevBin, args)
		if err != nil {
			log.Println("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		cleanup()
	}

	assert.NoError(nil)
}
