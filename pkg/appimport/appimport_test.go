package appimport_test

import (
	"path/filepath"
	"testing"

	"strings"

	"os"

	"github.com/ddev/ddev/pkg/appimport"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestValidateAsset tests validation of asset paths.
func TestValidateAsset(t *testing.T) {
	assert := asrt.New(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %s", err)
	}
	testdata := filepath.Join(cwd, "testdata")

	// test tilde expansion
	userDir, err := os.UserHomeDir()
	testDirName := "tmp.ddev.testpath-" + util.RandString(4)
	testDir := filepath.Join(userDir, testDirName)
	assert.NoError(err)
	err = os.Mkdir(testDir, 0755)
	assert.NoError(err)

	testPath, isArchive, err := appimport.ValidateAsset("~/"+testDirName, "files")
	assert.NoError(err)
	assert.Contains(testPath, userDir)
	assert.False(strings.Contains(testPath, "~"))
	assert.False(isArchive)
	err = os.Remove(testDir)
	assert.NoError(err)

	// test a relative path
	deepDir := filepath.Join(testdata, "dirlevel1", "dirlevel2")
	err = os.Chdir(deepDir)
	assert.NoError(err)
	testPath, _, err = appimport.ValidateAsset("../../dirlevel1", "files")
	assert.NoError(err)
	assert.Contains(testPath, "dirlevel1")

	// archive
	testArchivePath := filepath.Join(testdata, "somedb.sql.gz")
	_, isArchive, err = appimport.ValidateAsset(testArchivePath, "db")
	assert.NoError(err)
	assert.True(isArchive)

	// db no sql
	gofilePath := filepath.Join(testdata, "junk.go")
	_, isArchive, err = appimport.ValidateAsset(gofilePath, "db")
	if err != nil {
		assert.Contains(err.Error(), "provided path is not a .sql file or archive")
	}
	assert.Error(err)
	assert.False(isArchive)

	// files not a directory
	_, isArchive, err = appimport.ValidateAsset(gofilePath, "files")
	assert.False(isArchive)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "provided path is not a directory or archive")
	}
}
