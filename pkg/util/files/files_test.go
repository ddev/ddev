package files

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

var (
	temp            = os.TempDir()
	cwd             string
	testArchiveURL  = "https://github.com/drud/wordpress/releases/download/v0.1.0/files.tar.gz"
	testArchivePath = path.Join(os.TempDir(), "files.tar.gz")
)

func TestMain(m *testing.M) {
	err := system.DownloadFile(testArchivePath, testArchiveURL)
	if err != nil {
		log.Fatalf("archive download failed: %s", err)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}

	testRun := m.Run()

	os.Exit(testRun)
}

func TestUntargz(t *testing.T) {
	assert := assert.New(t)
	exDir := path.Join(temp, "extract")
	err := os.Mkdir(exDir, 0755)
	assert.NoError(err)

	err = Untargz(testArchivePath, exDir)
	assert.NoError(err)
	assert.True(system.FileExists(path.Join(exDir, "2017", "04", "pexels-photo-265186-100x100.jpeg")))
	assert.True(system.FileExists(path.Join(exDir, "2017", "04", "pexels-photo-265186-2000x1200.jpeg")))

	os.RemoveAll(exDir)
}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	assert := assert.New(t)
	dest := path.Join(temp, "testfile2")

	err := os.Chmod(testArchivePath, 0644)
	assert.NoError(err)

	err = CopyFile(testArchivePath, dest)
	assert.NoError(err)

	file, err := os.Stat(dest)
	assert.NoError(err)
	assert.Equal(int(file.Mode()), 0644)

	os.RemoveAll(dest)
}

// TestCopyDir tests copying a directory.
func TestCopyDir(t *testing.T) {
	assert := assert.New(t)
	dest := path.Join(temp, "copy")
	os.Mkdir(dest, 0755)

	// test source not a directory
	err := CopyDir(testArchivePath, temp)
	assert.Error(err)
	assert.Contains(err.Error(), "source is not a directory")

	// test destination exists
	err = CopyDir(temp, cwd)
	assert.Error(err)
	assert.Contains(err.Error(), "destination already exists")
	os.RemoveAll(dest)

	// copy a directory.
	err = CopyDir(cwd, dest)
	assert.NoError(err)
	assert.True(system.FileExists(path.Join(dest, "files.go")))
	assert.True(system.FileExists(path.Join(dest, "files_test.go")))

	os.RemoveAll(dest)
}
