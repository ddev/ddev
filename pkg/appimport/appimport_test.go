package appimport_test

import (
	"path/filepath"
	"testing"

	"strings"

	"os"

	"log"

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/util"
	gohomedir "github.com/mitchellh/go-homedir"
	asrt "github.com/stretchr/testify/assert"
)

// TestValidateAsset tests validation of asset paths.
func TestValidateAsset(t *testing.T) {
	assert := asrt.New(t)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}
	testdata := filepath.Join(cwd, "testdata")

	// test tilde expansion
	userDir, err := gohomedir.Dir()
	testDirName := "tmp.ddev.testpath-" + util.RandString(4)
	testDir := filepath.Join(userDir, testDirName)
	assert.NoError(err)
	err = os.Mkdir(testDir, 0755)
	assert.NoError(err)

	testPath, err := appimport.ValidateAsset("~/"+testDirName, "files")
	assert.NoError(err)
	assert.Contains(testPath, userDir)
	assert.False(strings.Contains(testPath, "~"))
	err = os.Remove(testDir)
	assert.NoError(err)

	// test a relative path
	deepDir := filepath.Join(testdata, "dirlevel1", "dirlevel2")
	err = os.Chdir(deepDir)
	assert.NoError(err)
	testPath, err = appimport.ValidateAsset("../../dirlevel1", "files")
	assert.NoError(err)

	assert.Contains(testPath, "dirlevel1")

	// archive
	testArchivePath := filepath.Join(testdata, "somedb.sql.gz")
	_, err = appimport.ValidateAsset(testArchivePath, "db")
	assert.Error(err)
	assert.Equal(err.Error(), "is archive")

	// db no sql
	gofilePath := filepath.Join(testdata, "junk.go")
	_, err = appimport.ValidateAsset(gofilePath, "db")
	assert.Contains(err.Error(), "provided path is not a .sql file or archive")
	assert.Error(err)

	// files not a directory
	_, err = appimport.ValidateAsset(gofilePath, "files")
	assert.Error(err)
	assert.Contains(err.Error(), "provided path is not a directory or archive")
}
