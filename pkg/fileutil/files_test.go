package fileutil_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
)

var testFileLocation = "testdata/regular_file"

// TestCopyDir tests copying a directory.
func TestCopyDir(t *testing.T) {
	assert := asrt.New(t)
	sourceDir := testcommon.CreateTmpDir("TestCopyDir_source")
	targetDir := testcommon.CreateTmpDir("TestCopyDir_target")

	subdir := filepath.Join(sourceDir, "some_content")
	err := os.Mkdir(subdir, 0755)
	assert.NoError(err)

	// test source not a directory
	err = fileutil.CopyDir(testFileLocation, sourceDir)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "source is not a directory")
	}

	// test destination exists
	err = fileutil.CopyDir(sourceDir, targetDir)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), "destination already exists")
	}
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

	err = fileutil.CopyDir(sourceDir, targetDir)
	assert.NoError(err)
	assert.True(fileutil.FileExists(filepath.Join(targetDir, "touch1.txt")))
	assert.True(fileutil.FileExists(filepath.Join(targetDir, "touch2.txt")))

	err = os.RemoveAll(sourceDir)
	assert.NoError(err)
	err = os.RemoveAll(targetDir)
	assert.NoError(err)

}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	assert := asrt.New(t)
	tmpTargetDir := testcommon.CreateTmpDir("TestCopyFile")
	tmpTargetFile := filepath.Join(tmpTargetDir, filepath.Base(testFileLocation))

	err := fileutil.CopyFile(testFileLocation, tmpTargetFile)
	assert.NoError(err)

	file, err := os.Stat(tmpTargetFile)
	assert.NoError(err)

	if err != nil {
		assert.False(file.IsDir())
	}
	err = os.RemoveAll(tmpTargetDir)
	assert.NoError(err)
}

// TestPurgeDirectory tests removal of directory contents without removing
// the directory itself.
func TestPurgeDirectory(t *testing.T) {
	assert := asrt.New(t)
	tmpPurgeDir := testcommon.CreateTmpDir("TestPurgeDirectory")
	tmpPurgeFile := filepath.Join(tmpPurgeDir, "regular_file")
	tmpPurgeSubFile := filepath.Join(tmpPurgeDir, "subdir", "regular_file")

	err := fileutil.CopyFile(testFileLocation, tmpPurgeFile)
	assert.NoError(err)

	err = os.Mkdir(filepath.Join(tmpPurgeDir, "subdir"), 0755)
	assert.NoError(err)

	err = fileutil.CopyFile(testFileLocation, tmpPurgeSubFile)
	assert.NoError(err)

	err = os.Chmod(tmpPurgeSubFile, 0444)
	assert.NoError(err)

	err = fileutil.PurgeDirectory(tmpPurgeDir)
	assert.NoError(err)

	assert.True(fileutil.FileExists(tmpPurgeDir))
	assert.False(fileutil.FileExists(tmpPurgeFile))
	assert.False(fileutil.FileExists(tmpPurgeSubFile))
}

// TestFgrepStringInFile tests the FgrepStringInFile utility function.
func TestFgrepStringInFile(t *testing.T) {
	assert := asrt.New(t)
	result, err := fileutil.FgrepStringInFile("testdata/fgrep_has_positive_contents.txt", "some needle we're looking for")
	assert.NoError(err)
	assert.True(result)
	result, err = fileutil.FgrepStringInFile("testdata/fgrep_negative_contents.txt", "some needle we're looking for")
	assert.NoError(err)
	assert.False(result)
}

// TestListFilesInDir makes sure the files we have in testfiles are properly enumerated
func TestListFilesInDir(t *testing.T) {
	assert := asrt.New(t)

	fileList, err := fileutil.ListFilesInDir("testdata/testfiles/")
	assert.NoError(err)
	assert.True(len(fileList) == 2)
	assert.Contains(fileList[0], "one.txt")
	assert.Contains(fileList[1], "two.txt")
}

// TestReplaceStringInFile tests the ReplaceStringInFile utility function.
func TestReplaceStringInFile(t *testing.T) {
	assert := asrt.New(t)
	tmp, err := ioutil.TempDir("", "")
	assert.NoError(err)
	newFilePath := filepath.Join(tmp, "newfile.txt")
	err = fileutil.ReplaceStringInFile("some needle we're looking for", "specialJUNKPattern", "testdata/fgrep_has_positive_contents.txt", newFilePath)
	assert.NoError(err)
	found, err := fileutil.FgrepStringInFile(newFilePath, "specialJUNKPattern")
	assert.NoError(err)
	assert.True(found)
}
