package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

// TestUntar tests untar functionality, including the starting directory
func TestUntar(t *testing.T) {
	assert := assert.New(t)
	exDir := testcommon.CreateTmpDir("TestUnTar1")

	err := util.Untar(TestArchivePath, exDir, "")
	assert.NoError(err)

	// Make sure that our base extraction directory is there
	finfo, err := os.Stat(filepath.Join(exDir, TestArchiveExtractDir))
	assert.NoError(err)
	assert.True(err == nil && finfo.IsDir())
	finfo, err = os.Stat(filepath.Join(exDir, TestArchiveExtractDir, ".ddev/config.yaml"))
	assert.NoError(err)
	assert.True(err == nil && !finfo.IsDir())

	err = os.RemoveAll(exDir)
	assert.NoError(err)

	// Now do the untar with an extraction root
	exDir = testcommon.CreateTmpDir("TestUnTar2")
	err = util.Untar(TestArchivePath, exDir, TestArchiveExtractDir)
	assert.NoError(err)

	finfo, err = os.Stat(filepath.Join(exDir, ".ddev"))
	assert.NoError(err)
	assert.True(err == nil && finfo.IsDir())
	finfo, err = os.Stat(filepath.Join(exDir, ".ddev/config.yaml"))
	assert.NoError(err)
	assert.True(err == nil && !finfo.IsDir())

	err = os.RemoveAll(exDir)
	assert.NoError(err)

}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	assert := assert.New(t)
	tmpTargetDir := testcommon.CreateTmpDir("TestCopyFile")
	tmpTargetFile := filepath.Join(tmpTargetDir, filepath.Base(TestArchivePath))

	err := util.CopyFile(TestArchivePath, tmpTargetFile)
	assert.NoError(err)

	file, err := os.Stat(tmpTargetFile)
	assert.NoError(err)

	if err != nil {
		assert.False(file.IsDir())
	}
	err = os.RemoveAll(tmpTargetDir)
	assert.NoError(err)
}

// TestCopyDir tests copying a directory.
func TestCopyDir(t *testing.T) {
	assert := assert.New(t)
	sourceDir := testcommon.CreateTmpDir("TestCopyDir_source")
	targetDir := testcommon.CreateTmpDir("TestCopyDir_target")

	subdir := filepath.Join(sourceDir, "some_content")
	err := os.Mkdir(subdir, 0755)
	assert.NoError(err)

	// test source not a directory
	err = util.CopyDir(TestArchivePath, sourceDir)
	assert.Error(err)
	assert.Contains(err.Error(), "source is not a directory")

	// test destination exists
	err = util.CopyDir(sourceDir, targetDir)
	assert.Error(err)
	assert.Contains(err.Error(), "destination already exists")
	err = os.RemoveAll(subdir)
	assert.NoError(err)

	// copy a directory and validate that we find files elsewhere
	err = os.RemoveAll(targetDir)
	assert.NoError(err)

	file, err := os.Create(filepath.Join(sourceDir, "touch1.txt"))
	assert.NoError(err)
	_ = file.Close()
	file, err = os.Create(filepath.Join(sourceDir, "touch2.txt"))
	assert.NoError(err)
	_ = file.Close()

	err = util.CopyDir(sourceDir, targetDir)
	assert.NoError(err)
	assert.True(system.FileExists(filepath.Join(targetDir, "touch1.txt")))
	assert.True(system.FileExists(filepath.Join(targetDir, "touch2.txt")))

	err = os.RemoveAll(sourceDir)
	assert.NoError(err)
	err = os.RemoveAll(targetDir)
	assert.NoError(err)

}
