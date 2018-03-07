package testcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
)

// TestTmpDir tests the ability to create a temporary directory.
func TestTmpDir(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory and ensure it exists.
	testDir := CreateTmpDir("TestTmpDir")
	dirStat, err := os.Stat(testDir)
	assert.NoError(err, "There is no error when getting directory details")
	assert.True(dirStat.IsDir(), "Temp Directory created and exists")

	// Clean up tempoary directory and ensure it no longer exists.
	CleanupDir(testDir)
	_, err = os.Stat(testDir)
	assert.Error(err, "Could not stat temporary directory")
	assert.True(os.IsNotExist(err), "Error is of type IsNotExists")
}

// TestChdir tests the Chdir function and ensures it will change to a temporary directory and then properly return
// to the original directory when cleaned up.
func TestChdir(t *testing.T) {
	assert := asrt.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err)

	// Create a temporary directory.
	testDir := CreateTmpDir("TestChdir")
	assert.NotEqual(startingDir, testDir, "Ensure our starting directory and temporary directory are not the same")

	// Change to the temporary directory.
	cleanupFunc := Chdir(testDir)
	currentDir, err := os.Getwd()
	assert.NoError(err)

	// On OSX this are created under /var, but /var is a symlink to /var/private, so we cannot ensure complete equality of these strings.
	assert.Contains(currentDir, testDir, "Ensure the current directory is the temporary directory we created")
	assert.True(reflect.TypeOf(cleanupFunc).Kind() == reflect.Func, "Chdir return is of type function")

	cleanupFunc()
	currentDir, err = os.Getwd()
	assert.NoError(err)
	assert.Equal(currentDir, startingDir, "Ensure we have changed back to the starting directory")

	CleanupDir(testDir)
}

// TestCaptureUserOut ensures capturing of stdout works as expected.
func TestCaptureUserOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := CaptureUserOut()
	text := util.RandString(128)
	output.UserOut.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestCaptureStdOut ensures capturing of stdout works as expected.
func TestCaptureStdOut(t *testing.T) {
	assert := asrt.New(t)
	restoreOutput := CaptureStdOut()
	text := util.RandString(128)
	fmt.Println(text)
	out := restoreOutput()

	assert.Contains(out, text)
}

// TestValidTestSite tests the TestSite struct behavior in the case of a valid configuration.
func TestValidTestSite(t *testing.T) {
	assert := asrt.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err, "Could not get current directory.")

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	ts := TestSite{
		Name:                          "TestValidTestSiteWordpress",
		SourceURL:                     "https://github.com/drud/wordpress/archive/v0.4.0.tar.gz",
		ArchiveInternalExtractionPath: "wordpress-0.4.0/",
		FilesTarballURL:               "https://github.com/drud/wordpress/releases/download/v0.4.0/files.tar.gz",
		DBTarURL:                      "https://github.com/drud/wordpress/releases/download/v0.4.0/db.tar.gz",
		Docroot:                       "htdocs",
		Type:                          "wordpress",
	}

	// Create a testsite and ensure the prepare() method extracts files into a temporary directory.
	err = ts.Prepare()
	if err != nil {
		t.Logf("Prepare() failed on TestSite %v, err=%v", ts, err)
		t.FailNow()
	}
	assert.NotNil(ts.Dir, "Directory is set.")
	docroot := filepath.Join(ts.Dir, ts.Docroot)
	dirStat, err := os.Stat(docroot)
	assert.NoError(err, "Docroot exists after prepare()")
	if err != nil {
		t.Fatalf("Directory did not exist after prepare(): %s", docroot)
	}
	assert.True(dirStat.IsDir(), "Docroot is a directory")

	cleanup := ts.Chdir()
	currentDir, err := os.Getwd()
	assert.NoError(err, "We can determine the current directory after changing to our TestSite directory")

	// On OSX this are created under /var, but /var is a symlink to /var/private, so we cannot ensure complete equality of these strings.
	assert.Contains(currentDir, ts.Dir, "Current directory matches expectations")

	cleanup()

	currentDir, err = os.Getwd()
	assert.NoError(err)
	assert.Equal(startingDir, currentDir, "Able to return to our original starting directory")

	ts.Cleanup()
	_, err = os.Stat(ts.Dir)
	assert.Error(err, "Could not stat temporary directory after cleanup")

}

// TestGetCachedArchive tests download and extraction of archives for test sites
// to testcache directory.
func TestGetCachedArchive(t *testing.T) {
	assert := asrt.New(t)

	sourceURL := "https://raw.githubusercontent.com/drud/ddev/master/.gitignore"
	exPath, archPath, err := GetCachedArchive("TestInvalidArchive", "test", "", sourceURL)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("archive extraction of %s failed", archPath))

	err = os.RemoveAll(filepath.Dir(exPath))
	assert.NoError(err)

	sourceURL = "http://invalid_domain/somefilethatdoesnotexists"
	exPath, archPath, err = GetCachedArchive("TestInvalidDownloadURL", "test", "", sourceURL)
	assert.Error(err)
	assert.Contains(err.Error(), fmt.Sprintf("Failed to download url=%s into %s", sourceURL, archPath))

	err = os.RemoveAll(filepath.Dir(exPath))
	assert.NoError(err)
}
