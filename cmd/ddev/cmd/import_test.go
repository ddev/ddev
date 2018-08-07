package cmd

import (
	"path/filepath"
	"testing"

	"os"

	"fmt"

	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/fileutil"
	gohomedir "github.com/mitchellh/go-homedir"
	asrt "github.com/stretchr/testify/assert"
)

// TestImportTilde tests passing paths to import-files that use ~ to represent home dir.
func TestImportTildeArchives(t *testing.T) {
	assert := asrt.New(t)

	testFiles := []string{
		"testfile.tar.gz",
		"testfile.tar",
		"testfile.zip",
	}

	for _, site := range DevTestSites {
		for _, testFile := range testFiles {
			homedir, err := gohomedir.Dir()
			assert.NoError(err)
			cwd, _ := os.Getwd()
			testFilePath := filepath.Join(homedir, testFile)
			err = fileutil.CopyFile(filepath.Join(cwd, "testdata", testFile), testFilePath)
			assert.NoError(err)

			cleanup := site.Chdir()
			defer rmFile(testFilePath)

			// this ~ should be expanded by shell
			args := []string{"import-files", "--src", fmt.Sprintf("~/%s", testFile)}
			out, err := exec.RunCommand(DdevBin, args)
			if err != nil {
				t.Log("Error Output from ddev import-files:", out, site)
			}
			assert.NoError(err)
			assert.Contains(string(out), "Successfully imported files")

			// this ~ is not expanded by shell, ddev should convert it to a valid path
			args = []string{"import-files", fmt.Sprintf("--src=~/%s", testFile)}
			out, err = exec.RunCommand(DdevBin, args)
			if err != nil {
				t.Log("Error Output from ddev import-files:", out, site)
			}
			assert.NoError(err)
			assert.Contains(string(out), "Successfully imported files")

			cleanup()
		}
	}

	assert.NoError(nil)
}

func TestImportTildeDir(t *testing.T) {
	assert := asrt.New(t)

	testDir := "testdir"

	for _, site := range DevTestSites {
		homedir, err := gohomedir.Dir()
		assert.NoError(err)
		cwd, _ := os.Getwd()
		testDirPath := filepath.Join(homedir, testDir)
		err = fileutil.CopyDir(filepath.Join(cwd, "testdata", testDir), testDirPath)
		assert.NoError(err)

		cleanup := site.Chdir()
		defer rmDir(testDirPath)

		// this ~ should be expanded by shell
		args := []string{"import-files", "--src", fmt.Sprintf("~/%s", testDir)}
		out, err := exec.RunCommand(DdevBin, args)
		if err != nil {
			t.Log("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		// this ~ is not expanded by shell, ddev should convert it to a valid path
		args = []string{"import-files", fmt.Sprintf("--src=~/%s", testDir)}
		out, err = exec.RunCommand(DdevBin, args)
		if err != nil {
			t.Log("Error Output from ddev import-files:", out, site)
		}
		assert.NoError(err)
		assert.Contains(string(out), "Successfully imported files")

		cleanup()
	}

	assert.NoError(nil)
}

// rmFile simply allows us to defer os.Remove while ignoring the error return.
func rmFile(fullPath string) {
	_ = os.Remove(fullPath)
}

// rmDir allows us to defer os.RemoveAll while ignoring the error return
func rmDir(fullPath string) {
	_ = os.RemoveAll(fullPath)
}
