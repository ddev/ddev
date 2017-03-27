package testcommon

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateTmpDir tests the functionality that is called when "ddev start" is executed
func TestTmpDir(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory and ensure it exists.
	testDir := CreateTmpDir()
	dirStat, err := os.Stat(testDir)
	assert.NoError(err, "There is no error when getting directory details")
	assert.True(dirStat.IsDir(), "Temp Directory created and exists")

	// Clean up tempoary directory and ensure it no longer exists.
	err = CleanupDir(testDir)
	assert.NoError(err, "Clean up temporary directory")
	dirStat, err = os.Stat(testDir)
	assert.Error(err, "Could not stat temporary directory")
	assert.True(os.IsNotExist(err), "Error is of type IsNotExists")
}

// TestCreateTmpDir tests the functionality that is called when "ddev start" is executed
func TestChDir(t *testing.T) {
	assert := assert.New(t)
	// Get the current working directory.
	startingDir, err := os.Getwd()
	assert.NoError(err)

	// Create a temporary directory.
	testDir := CreateTmpDir()
	assert.NotEqual(startingDir, testDir, "Ensure our starting directory and temporary directory are not the same")

	// Change to the temporary directory.
	cleanupFunc := Chdir(testDir)
	currentDir, err := os.Getwd()
	assert.Equal(testDir, currentDir, "Ensure the current directory is the temporary directory we created")
	assert.True(reflect.TypeOf(cleanupFunc).Kind() == reflect.Func, "Chdir return is of type function")

	cleanupFunc()
	currentDir, err = os.Getwd()
	assert.NoError(err)
	assert.Equal(currentDir, startingDir, "Ensure we have changed back to the starting directory")

	err = CleanupDir(testDir)
	assert.NoError(err, "Clean up test directory")
}
