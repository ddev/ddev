package files

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/drud/drud-go/utils/system"
	"github.com/stretchr/testify/assert"
)

var (
	temp     = os.TempDir()
	cwd      string
	testFile = path.Join(temp, "testfile")
)

func TestMain(m *testing.M) {
	tf, err := os.Create(testFile)
	if err != nil {
		log.Fatalf("failed to create test file: %s", err)
	}
	tf.Close()

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %s", err)
	}

	testRun := m.Run()

	os.Exit(testRun)
}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	assert := assert.New(t)
	dest := path.Join(temp, "testfile2")

	err := os.Chmod(testFile, 0644)
	assert.NoError(err)

	err = CopyFile(testFile, dest)
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
	err := CopyDir(testFile, temp)
	assert.Error(err)
	msg := fmt.Sprintf("%v", err)
	assert.Contains(msg, "source is not a directory")

	// test destination exists
	err = CopyDir(temp, cwd)
	assert.Error(err)
	msg = fmt.Sprintf("%v", err)
	assert.Contains(msg, "destination already exists")
	os.RemoveAll(dest)

	// copy a directory.
	err = CopyDir(cwd, dest)
	assert.NoError(err)
	assert.True(system.FileExists(path.Join(dest, "files.go")))
	assert.True(system.FileExists(path.Join(dest, "files_test.go")))

	os.RemoveAll(dest)
}
