package testcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTmpDir tests the ability to create a temporary directory.
func TestTmpDir(t *testing.T) {
	assert := assert.New(t)

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
	assert := assert.New(t)
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

// TestCaptureStdOut ensures capturing of standard out works as expected.
func TestCaptureStdOut(t *testing.T) {
	assert := assert.New(t)
	restoreOutput := CaptureStdOut()
	text := RandString(128)
	fmt.Print(text)
	out := restoreOutput()

	assert.Equal(text, out)
}

// TestValidTestSite tests the TestSite struct behavior in the case of a valid configuration.
func TestValidTestSite(t *testing.T) {
	assert := assert.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err, "Could not get current directory.")

	// It's not ideal to copy/paste this archive around, but we don't actually care about the contents
	// of the archive for this test, only that it exists and can be extracted. This should (knock on wood)
	//not need to be updated over time.
	ts := TestSite{
		Name:      "TestValidTestSiteDrupal8",
		SourceURL: "https://github.com/drud/drupal8/archive/v0.5.0.tar.gz",
	}

	// Create a testsite and ensure the prepare() method extracts files into a temporary directory.
	err = ts.Prepare()
	if err != nil {
		t.Logf("Prepare() failed on TestSite %v, err=%v", ts, err)
		t.FailNow()
	}
	assert.NotNil(ts.Dir, "Directory is set.")
	docroot := filepath.Join(ts.Dir, "docroot")
	dirStat, err := os.Stat(docroot)
	assert.NoError(err, "Docroot exists after prepare()")
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

// TestInvalidTestSite ensures that errors are returned in cases where Prepare() can't download or extract an archive.
func TestInvalidTestSite(t *testing.T) {
	assert := assert.New(t)

	testSites := []TestSite{
		// This should generate a 404 page on github, which will be downloaded, but cannot be extracted (as it's not a true tar.gz)
		TestSite{
			Name:      "TestInvalidTestSite404",
			SourceURL: "https://github.com/drud/drupal8/archive/somevaluethatdoesnotexist.tar.gz",
		},
		// This is an invalid domain, so it can't even be downloaded. This tests error handling in the case of
		// a site URL which does not exist
		TestSite{
			Name:      "TestInvalidTestSiteInvalidDomain",
			SourceURL: "http://invalid_domain/somefilethatdoesnotexists",
		},
	}

	for i := range testSites {
		ts := testSites[i]
		// Create a testsite and ensure the prepare() method extracts files into a temporary directory.
		err := ts.Prepare()
		assert.Error(err, "ts.Prepare() fails because of missing config.yml or untar failure")
	}
}

// TestRandString ensures that RandString only generates string of the correct value and characters.
func TestRandString(t *testing.T) {
	assert := assert.New(t)
	stringLengths := []int{2, 4, 8, 16, 23, 47}

	for _, stringLength := range stringLengths {
		testString := RandString(stringLength)
		assert.Equal(len(testString), stringLength, fmt.Sprintf("Generated string is of length %d", stringLengths))
	}

	lb := "a"
	setLetterBytes(lb)
	testString := RandString(1)
	assert.Equal(testString, lb)
}
