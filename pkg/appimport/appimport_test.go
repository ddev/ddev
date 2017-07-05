package appimport_test

import (
	"path/filepath"
	"testing"

	"strings"

	"os"

	"log"

	"github.com/drud/ddev/pkg/appimport"
	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

// TestValidateAsset tests validation of asset paths.
func TestValidateAsset(t *testing.T) {
	assert := assert.New(t)

	testArchivePath := filepath.Join(testcommon.CreateTmpDir("appimport"), "db.tar.gz")

	testFile, err := os.Create(testArchivePath)
	if err != nil {
		log.Fatalf("failed to create dummy test file: %v", err)
	}
	err = testFile.Close()
	if err != nil {
		log.Fatalf("failed to create dummy test file: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}

	// test tilde expansion
	userDir, err := homedir.Dir()
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
	testPath, err = appimport.ValidateAsset("../../vendor", "files")
	assert.NoError(err)
	upTwo := strings.TrimSuffix(cwd, "/pkg/appimport")
	assert.Contains(testPath, upTwo)

	// archive
	_, err = appimport.ValidateAsset(testArchivePath, "db")
	assert.Error(err)
	assert.Equal(err.Error(), "is archive")

	// db no sql
	_, err = appimport.ValidateAsset("appimport.go", "db")
	assert.Contains(err.Error(), "provided path is not a .sql file or archive")
	assert.Error(err)

	// files not a directory
	_, err = appimport.ValidateAsset("appimport.go", "files")
	assert.Error(err)
	assert.Contains(err.Error(), "provided path is not a directory or archive")

	err = os.RemoveAll(filepath.Dir(testArchivePath))
	util.CheckErr(err)
}
