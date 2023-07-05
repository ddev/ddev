package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemoveAllExcept tests copying a directory.
func TestRemoveAllExcept(t *testing.T) {
	testDir, _ := os.Getwd()
	assert := assert.New(t)
	require := require.New(t)

	sourceDir := filepath.Join(testDir, "testdata", "remove_all_except")
	targetBaseDir := testcommon.CreateTmpDir("TestRemoveAllExcept")
	targetDir := filepath.Join(targetBaseDir, "testdata")
	defer os.RemoveAll(targetBaseDir)

	err := fileutil.CopyDir(sourceDir, targetDir)
	assert.NoError(err)

	err = fileutil.RemoveAllExcept(targetDir, []string{"keep/*", "keep_partial", "sub/keep/*"})
	assert.NoError(err)

	require.DirExists(targetDir)

	assert.DirExists(filepath.Join(targetDir, "keep"))
	assert.DirExists(filepath.Join(targetDir, "keep", "keep_sub"))
	assert.FileExists(filepath.Join(targetDir, "keep", "keep.txt"))
	assert.FileExists(filepath.Join(targetDir, "keep", "keep_sub", "keep.txt"))

	assert.DirExists(filepath.Join(targetDir, "keep_partial"))
	assert.NoDirExists(filepath.Join(targetDir, "keep_partial", "remove"))
	assert.NoFileExists(filepath.Join(targetDir, "keep_partial", "remove.txt"))
	assert.NoFileExists(filepath.Join(targetDir, "keep_partial", "remove", "remove.txt"))

	assert.DirExists(filepath.Join(targetDir, "sub", "keep"))
	assert.DirExists(filepath.Join(targetDir, "sub", "keep", "keep_sub"))
	assert.FileExists(filepath.Join(targetDir, "sub", "keep", "keep.txt"))
	assert.FileExists(filepath.Join(targetDir, "sub", "keep", "keep_sub", "keep.txt"))

	assert.NoFileExists(filepath.Join(targetDir, "remove.txt"))

	assert.NoDirExists(filepath.Join(targetDir, "remove"))
	assert.NoFileExists(filepath.Join(targetDir, "remove", "remove.txt"))
}
