package util_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/testcommon"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

var (
	testArchiveURL        = "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz"
	testArchiveExtractDir = "wordpress-0.4.0/"
	testArchivePath       string
)

func TestMain(m *testing.M) {
	testPath, err := ioutil.TempDir("", "filetest")
	util.CheckErr(err)
	testPath, err = filepath.EvalSymlinks(testPath)
	util.CheckErr(err)
	testPath = filepath.Clean(testPath)
	testArchivePath = filepath.Join(testPath, "files.tar.gz")

	err = system.DownloadFile(testArchivePath, testArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	testRun := m.Run()

	os.Exit(testRun)
}

// TestUntar tests untar functionality, including the starting directory
func TestUntar(t *testing.T) {
	assert := assert.New(t)
	exDir := testcommon.CreateTmpDir("TestUnTar1")

	err := util.Untar(testArchivePath, exDir, "")
	assert.NoError(err)

	// Make sure that our base extraction directory is there
	finfo, err := os.Stat(filepath.Join(exDir, testArchiveExtractDir))
	assert.NoError(err)
	assert.True(err == nil && finfo.IsDir())
	finfo, err = os.Stat(filepath.Join(exDir, testArchiveExtractDir, ".ddev/config.yaml"))
	assert.NoError(err)
	assert.True(err == nil && !finfo.IsDir())

	err = os.RemoveAll(exDir)
	assert.NoError(err)

	// Now do the untar with an extraction root
	exDir = testcommon.CreateTmpDir("TestUnTar2")
	err = util.Untar(testArchivePath, exDir, testArchiveExtractDir)
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
	temp := testcommon.CreateTmpDir("TestCopyFile")

	dest := filepath.Join(temp, "testfile2")

	err := os.Chmod(testArchivePath, 0644)
	assert.NoError(err)

	err = util.CopyFile(testArchivePath, dest)
	assert.NoError(err)

	file, err := os.Stat(dest)
	assert.NoError(err)
	assert.Equal(int(file.Mode()), 0644)

	err = os.RemoveAll(dest)
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
	err = util.CopyDir(testArchivePath, sourceDir)
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
