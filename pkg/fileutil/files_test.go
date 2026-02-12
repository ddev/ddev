package fileutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.Contains(err.Error(), fmt.Sprintf("CopyDir: source directory %s is not a directory", filepath.Join(testFileLocation)))
	}

	// test destination exists
	err = fileutil.CopyDir(sourceDir, targetDir)
	assert.Error(err)
	if err != nil {
		assert.Contains(err.Error(), fmt.Sprintf("CopyDir: destination %s already exists", filepath.Join(targetDir)))
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

	err = util.Chmod(tmpPurgeSubFile, 0444)
	assert.NoError(err)

	err = fileutil.PurgeDirectory(tmpPurgeDir)
	assert.NoError(err)

	assert.True(fileutil.FileExists(tmpPurgeDir))
	assert.False(fileutil.FileExists(tmpPurgeFile))
	assert.False(fileutil.FileExists(tmpPurgeSubFile))
}

// TestPurgeDirectoryExcept tests removal of directory contents except specified files.
func TestPurgeDirectoryExcept(t *testing.T) {
	dir := t.TempDir()

	// Create files: README.txt plus some others
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.txt"), []byte("keep me"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.yaml"), []byte("remove me"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project1.yaml"), []byte("remove me too"), 0644))

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, 3, len(files))

	err = fileutil.PurgeDirectoryExcept(dir, map[string]bool{"README.txt": true})
	require.NoError(t, err)

	filesAfter, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Equal(t, 1, len(filesAfter))
	require.Equal(t, "README.txt", filesAfter[0].Name())

	// Verify preserved file content is intact
	content, err := os.ReadFile(filepath.Join(dir, "README.txt"))
	require.NoError(t, err)
	require.Equal(t, "keep me", string(content))
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

// TestListFilesInDirNoSubdirsFullPath tests ListFilesInDirNoSubdirsFullPath()
func TestListFilesInDirNoSubdirsFullPath(t *testing.T) {
	fileList, err := fileutil.ListFilesInDirFullPath(filepath.Join("testdata", t.Name()), true)
	require.NoError(t, err)
	require.Len(t, fileList, 2)
	require.Contains(t, fileList[0], "one.txt")
	require.Contains(t, fileList[1], "two.txt")
}

// TestReplaceStringInFile tests the ReplaceStringInFile utility function.
func TestReplaceStringInFile(t *testing.T) {
	assert := asrt.New(t)
	tmpDir := testcommon.CreateTmpDir(t.Name())
	newFilePath := filepath.Join(tmpDir, "newfile.txt")
	err := fileutil.ReplaceStringInFile("some needle we're looking for", "specialJUNKPattern", "testdata/fgrep_has_positive_contents.txt", newFilePath)
	assert.NoError(err)
	found, err := fileutil.FgrepStringInFile(newFilePath, "specialJUNKPattern")
	assert.NoError(err)
	assert.True(found)
}

// TestFindSimulatedXsymSymlinks tests FindSimulatedXsymSymlinks
func TestFindSimulatedXsymSymlinks(t *testing.T) {
	assert := asrt.New(t)
	testDir, _ := os.Getwd()
	targetDir := filepath.Join(testDir, "testdata", "symlinks")
	links, err := fileutil.FindSimulatedXsymSymlinks(targetDir)
	assert.NoError(err)
	assert.Len(links, 8)
}

// TestReplaceSimulatedXsymSymlinks tries a number of symlinks to make
// sure we can parse and replace symlinks.
func TestReplaceSimulatedXsymSymlinks(t *testing.T) {
	testDir, _ := os.Getwd()
	assert := asrt.New(t)
	if nodeps.IsWindows() && !fileutil.CanCreateSymlinks() {
		t.Skip("Skipping on Windows because test machine can't create symlnks")
	}
	sourceDir := filepath.Join(testDir, "testdata", "symlinks")
	targetDir := testcommon.CreateTmpDir("TestReplaceSimulated")
	//nolint: errcheck
	defer os.RemoveAll(targetDir)
	err := os.Chdir(targetDir)
	assert.NoError(err)

	// Make sure we leave the testDir as we found it..
	//nolint: errcheck
	defer os.Chdir(testDir)
	// CopyDir skips real symlinks, but we only care about simulated ones, so it's OK
	err = fileutil.CopyDir(sourceDir, filepath.Join(targetDir, "symlinks"))
	assert.NoError(err)
	links, err := fileutil.FindSimulatedXsymSymlinks(targetDir)
	assert.NoError(err)
	assert.Len(links, 8)
	err = fileutil.ReplaceSimulatedXsymSymlinks(links)
	assert.NoError(err)

	for _, link := range links {
		fi, err := os.Stat(link.LinkLocation)
		assert.NoError(err)
		linkFi, err := os.Lstat(link.LinkLocation)
		assert.NoError(err)
		if err == nil && fi != nil && !fi.IsDir() {
			// Read the symlink as a file. It should resolve with the actual content of target
			contents, err := os.ReadFile(link.LinkLocation)
			assert.NoError(err)
			expectedContent := "textfile " + filepath.Base(link.LinkTarget) + "\n"
			assert.Equal(expectedContent, string(contents))
		}
		// Now stat the link and make sure it's a link and points where it should
		if linkFi.Mode()&os.ModeSymlink != 0 {
			targetFile, err := os.Readlink(link.LinkLocation)
			assert.NoError(err)
			assert.Equal(link.LinkTarget, targetFile)
			_ = targetFile
		}
	}
}

// TestIsSameFile tests the IsSameFile utility function.
func TestIsSameFile(t *testing.T) {
	assert := asrt.New(t)

	tmpDir, err := testcommon.OsTempDir()
	assert.NoError(err)

	dirSymlink, err := filepath.Abs(filepath.Join(tmpDir, fileutil.RandomFilenameBase()))
	require.NoError(t, err)
	testdataAbsolute, err := filepath.Abs("testdata")
	require.NoError(t, err)
	err = os.Symlink(testdataAbsolute, dirSymlink)
	assert.NoError(err)
	//nolint: errcheck
	defer os.Remove(dirSymlink)

	// At this point, dirSymLink and "testdata" should be equivalent
	isSame, err := fileutil.IsSameFile("testdata", dirSymlink)
	assert.NoError(err)
	assert.True(isSame)
	// Test with files that are equivalent (through symlink)
	isSame, err = fileutil.IsSameFile("testdata/testfiles/one.txt", filepath.Join(dirSymlink, "testfiles", "one.txt"))
	assert.NoError(err)
	assert.True(isSame)

	// Test files that are *not* equivalent.
	isSame, err = fileutil.IsSameFile("testdata/testfiles/one.txt", "testdata/testfiles/two.txt")
	assert.NoError(err)
	assert.False(isSame)
}

// TestRemoveFilesMatchingGlob tests that RemoveFilesMatchingGlob works
func TestRemoveFilesMatchingGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a few test files
	files := []string{"match1.log", "match2.log", "keep.txt"}
	for _, name := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	// Call RemoveFilesMatchingGlob to remove *.log files
	err := fileutil.RemoveFilesMatchingGlob(filepath.Join(tmpDir, "*.log"))
	if err != nil {
		t.Fatalf("RemoveFilesMatchingGlob returned error: %v", err)
	}

	// Assert that .log files are gone and .txt remains
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		_, err := os.Stat(path)
		if filepath.Ext(name) == ".log" {
			if !os.IsNotExist(err) {
				t.Errorf("expected file %s to be removed, but it exists", path)
			}
		} else {
			if err != nil {
				t.Errorf("expected file %s to exist, but got error: %v", path, err)
			}
		}
	}

	// It should not error when nothing matches
	err = fileutil.RemoveFilesMatchingGlob(filepath.Join(tmpDir, "*.nomatch"))
	if err != nil {
		t.Errorf("expected no error when no files match, got: %v", err)
	}
}

// TestCopyFilesMatchingGlob tests that CopyFilesMatchingGlob works
func TestCopyFilesMatchingGlob(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create test files in source directory
	files := []string{"config1.yaml", "config2.yaml", "readme.txt", "notes.md"}
	for _, name := range files {
		err := os.WriteFile(filepath.Join(srcDir, name), []byte("content of "+name), 0644)
		require.NoError(t, err)
	}

	// Create a subdirectory named "subdir.yaml" to ensure directories are skipped
	err := os.Mkdir(filepath.Join(srcDir, "subdir.yaml"), 0755)
	require.NoError(t, err)

	// Copy only *.yaml files
	copiedFiles, err := fileutil.CopyFilesMatchingGlob(srcDir, destDir, "*.yaml")
	require.NoError(t, err)
	require.Len(t, copiedFiles, 2)
	require.Contains(t, copiedFiles, "config1.yaml")
	require.Contains(t, copiedFiles, "config2.yaml")

	// Verify the copied files exist and have correct content
	for _, name := range []string{"config1.yaml", "config2.yaml"} {
		destPath := filepath.Join(destDir, name)
		require.True(t, fileutil.FileExists(destPath), "expected %s to exist in dest", name)
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		require.Equal(t, "content of "+name, string(content))
	}

	// Verify non-matching files were not copied
	for _, name := range []string{"readme.txt", "notes.md", "subdir.yaml"} {
		destPath := filepath.Join(destDir, name)
		require.False(t, fileutil.FileExists(destPath), "expected %s to not exist in dest", name)
	}

	// Test with no matches - should return empty slice with no error
	emptyDest := t.TempDir()
	copiedFiles, err = fileutil.CopyFilesMatchingGlob(srcDir, emptyDest, "*.nomatch")
	require.NoError(t, err)
	require.Len(t, copiedFiles, 0)
}

// TestCheckSignatureOrNoFile tests the CheckSignatureOrNoFile function
// including the correct handling of empty files per commit 986812445
func TestCheckSignatureOrNoFile(t *testing.T) {
	assert := asrt.New(t)
	tmpDir := testcommon.CreateTmpDir(t.Name())
	defer os.RemoveAll(tmpDir)

	signature := nodeps.DdevFileSignature

	// Test 1: Non-existent file should return nil (can overwrite)
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")
	err := fileutil.CheckSignatureOrNoFile(nonExistentFile, signature)
	assert.NoError(err, "non-existent file should be safe to overwrite")

	// Test 2: File with signature should return nil (can overwrite)
	fileWithSig := filepath.Join(tmpDir, "with_signature.txt")
	err = os.WriteFile(fileWithSig, []byte(signature+"\nsome content"), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(fileWithSig, signature)
	assert.NoError(err, "file with signature should be safe to overwrite")

	// Test 3: Empty file should return nil (can overwrite)
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(emptyFile, signature)
	assert.NoError(err, "empty file should be safe to overwrite")

	// Test 4: File without signature and with content should return error (cannot overwrite)
	fileWithoutSig := filepath.Join(tmpDir, "without_signature.txt")
	err = os.WriteFile(fileWithoutSig, []byte("user content without signature"), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(fileWithoutSig, signature)
	assert.Error(err, "file without signature should not be safe to overwrite")
	if err != nil {
		assert.Contains(err.Error(), "signature was not found")
	}

	// Test 5: Directory with all files having signature should return nil
	dirWithSig := filepath.Join(tmpDir, "dir_with_sig")
	err = os.Mkdir(dirWithSig, 0755)
	assert.NoError(err)
	file1 := filepath.Join(dirWithSig, "file1.txt")
	err = os.WriteFile(file1, []byte(signature+"\ncontent1"), 0644)
	assert.NoError(err)
	file2 := filepath.Join(dirWithSig, "file2.txt")
	err = os.WriteFile(file2, []byte(signature+"\ncontent2"), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(dirWithSig, signature)
	assert.NoError(err, "directory with all files having signature should be safe to overwrite")

	// Test 6: Directory with one file without signature should return error
	dirMixed := filepath.Join(tmpDir, "dir_mixed")
	err = os.Mkdir(dirMixed, 0755)
	assert.NoError(err)
	fileGood := filepath.Join(dirMixed, "good.txt")
	err = os.WriteFile(fileGood, []byte(signature+"\ncontent"), 0644)
	assert.NoError(err)
	fileBad := filepath.Join(dirMixed, "bad.txt")
	err = os.WriteFile(fileBad, []byte("user content"), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(dirMixed, signature)
	assert.Error(err, "directory with file without signature should not be safe to overwrite")

	// Test 7: Directory with empty file should be safe to overwrite
	dirWithEmpty := filepath.Join(tmpDir, "dir_with_empty")
	err = os.Mkdir(dirWithEmpty, 0755)
	assert.NoError(err)
	emptyInDir := filepath.Join(dirWithEmpty, "empty.txt")
	err = os.WriteFile(emptyInDir, []byte(""), 0644)
	assert.NoError(err)
	err = fileutil.CheckSignatureOrNoFile(dirWithEmpty, signature)
	assert.NoError(err, "directory with empty file should be safe to overwrite")
}
