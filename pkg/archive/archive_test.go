package archive_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnarchive tests unzip/tar/tar.gz/tgz functionality, including the starting extraction-skip directory
func TestUnarchive(t *testing.T) {
	// testUnarchiveDir is the directory we may want to use to start extracting.
	testUnarchiveDir := "dir2/"

	assert := asrt.New(t)

	for _, suffix := range []string{"zip", "tar", "tar.gz", "tgz"} {
		source := filepath.Join("testdata", t.Name(), "testfile"+"."+suffix)
		exDir := testcommon.CreateTmpDir("testfile" + suffix)

		// default function to untar
		unarchiveFunc := archive.Untar
		if suffix == "zip" {
			unarchiveFunc = archive.Unzip
		}

		err := unarchiveFunc(source, exDir, "")
		assert.NoError(err)

		// Make sure that our base extraction directory is there
		finfo, err := os.Stat(filepath.Join(exDir, testUnarchiveDir))
		assert.NoError(err)
		assert.True(err == nil && finfo.IsDir())
		finfo, err = os.Stat(filepath.Join(exDir, testUnarchiveDir, "dir2_file.txt"))
		assert.NoError(err)
		assert.True(err == nil && !finfo.IsDir())

		err = os.RemoveAll(exDir)
		assert.NoError(err)

		// Now do the unarchive with an extraction root
		exDir = testcommon.CreateTmpDir("testfile" + suffix + "2")

		err = unarchiveFunc(source, exDir, testUnarchiveDir)
		assert.NoError(err)

		// Only the dir2_file should remain
		finfo, err = os.Stat(filepath.Join(exDir, "dir2_file.txt"))
		assert.NoError(err)
		assert.True(err == nil && !finfo.IsDir())

		err = os.RemoveAll(exDir)
		assert.NoError(err)
	}
}

// TestArchiveTar tests creation of a simple tarball
func TestArchiveTar(t *testing.T) {
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	tmpDir := testcommon.CreateTmpDir(t.Name())
	tarballFile, err := os.CreateTemp(tmpDir, t.Name()+"_*.tar.gz")
	require.NoError(t, err)

	tarSrc := filepath.Join(origDir, "testdata", t.Name())
	err = os.Chdir(tarSrc)
	require.NoError(t, err)

	expectations := map[string]fs.FileMode{}
	for _, f := range []string{".test.sh", "root.txt", filepath.Join("subdir1", "subdir1.txt")} {
		fi, err := os.Stat(f)
		assert.NoError(err)
		expectations[f] = fi.Mode()
	}

	err = archive.Tar(tarSrc, tarballFile.Name(), filepath.Join("subdir1", "subdir2"))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		_ = tarballFile.Close()

		_ = os.Remove(tarballFile.Name())
		_ = os.RemoveAll(tmpDir)
	})

	_ = os.Chdir(tmpDir)
	err = archive.Untar(tarballFile.Name(), tmpDir, "")
	require.NoError(t, err)

	for fileName, mode := range expectations {
		testedFileName, err := filepath.Abs(fileName)
		require.NoError(t, err, "fileName err: %v %v", testedFileName, err)
		fi, err := os.Stat(fileName)
		require.NoError(t, err)
		require.NotNil(t, fi)
		//desc := fmt.Sprintf("%s: Orig mode=%o, found mode=%o", fileName, mode, fi.Mode())
		//t.Log(desc)
		require.Equal(t, fi.Mode(), mode, "expected mode for %s was %o but got %o", fileName, mode, fi.Mode())
	}
	require.NoFileExists(t, filepath.Join(tmpDir, "subdir1", "subdir2", "s2.txt"))
}

// TestArchiveTarGz tests creation of a simple gzipped tarball
func TestArchiveTarGz(t *testing.T) {
	assert := asrt.New(t)
	pwd, _ := os.Getwd()
	tarballFile, err := os.CreateTemp("", t.Name()+"*.tar.gz")
	assert.NoError(err)

	err = archive.Tar(filepath.Join(pwd, "testdata", t.Name()), tarballFile.Name(), filepath.Join("subdir1", "subdir2"))
	assert.NoError(err)

	tmpDir := testcommon.CreateTmpDir(t.Name())

	t.Cleanup(
		func() {
			_ = tarballFile.Close()
			_ = os.Remove(tarballFile.Name())
			_ = os.RemoveAll(tmpDir)
		})
	err = archive.Untar(tarballFile.Name(), tmpDir, "")
	assert.NoError(err)

	assert.FileExists(filepath.Join(tmpDir, "root.txt"))
	assert.FileExists(filepath.Join(tmpDir, "subdir1", "subdir1.txt"))
	assert.NoFileExists(filepath.Join(tmpDir, "subdir1", "subdir2", "s2.txt"))
}

// TestExtractTarballWithCleanup tests ExtractTarballWithCleanup
func TestExtractTarballWithCleanup(t *testing.T) {
	assert := asrt.New(t)

	for _, suffix := range []string{"tar", "tar.gz", "tgz"} {
		tarball := path.Join("testdata", t.Name(), "testfile"+"."+suffix)
		dir, cleanup, err := archive.ExtractTarballWithCleanup(tarball, false)
		assert.NoError(err)
		assert.DirExists(dir)
		assert.FileExists(path.Join(dir, "dir1/dir1_file.txt"))
		cleanup()
		assert.NoDirExists(dir)

		dir, cleanup, err = archive.ExtractTarballWithCleanup(tarball, true)
		assert.NoError(err)
		assert.DirExists(dir)
		assert.FileExists(path.Join(dir, "dir1_file.txt"))
		cleanup()
		assert.NoDirExists(dir)
	}
}

// TestDownloadAndExtractTarball tests DownloadAndExtractTarball
func TestDownloadAndExtractTarball(t *testing.T) {
	testTarball := "https://github.com/ddev/ddev-drupal-solr/archive/refs/tags/v1.2.3.tar.gz"

	dir, cleanup, err := archive.DownloadAndExtractTarball(testTarball, true)
	if cleanup != nil {
		defer cleanup()
	}
	require.NoError(t, err)
	require.DirExists(t, dir)
	require.FileExists(t, path.Join(dir, "install.yaml"))
	cleanup()
	require.NoDirExists(t, dir)
}

// TestUntarSymlinks tests that symlinks are properly extracted from tarballs
func TestUntarSymlinks(t *testing.T) {
	assert := asrt.New(t)

	// Create a temporary directory with a file and a symlink
	srcDir := testcommon.CreateTmpDir(t.Name() + "_src")
	t.Cleanup(func() {
		_ = os.RemoveAll(srcDir)
	})

	// Create a test file
	testFile := filepath.Join(srcDir, "target.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create a symlink in the root pointing to the file
	symlinkPath := filepath.Join(srcDir, "link_to_target.txt")
	err = os.Symlink("target.txt", symlinkPath)
	require.NoError(t, err)

	// Create a symlink in subdir pointing to parent file
	symlinkInSubdir := filepath.Join(subDir, "link_to_parent.txt")
	err = os.Symlink("../target.txt", symlinkInSubdir)
	require.NoError(t, err)

	// Create tarball
	tarballFile, err := os.CreateTemp("", t.Name()+"_*.tar.gz")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tarballFile.Close()
		_ = os.Remove(tarballFile.Name())
	})

	err = archive.Tar(srcDir, tarballFile.Name(), "")
	require.NoError(t, err)

	// Verify tarball contents contain proper symlink entries
	tf, err := os.Open(tarballFile.Name())
	require.NoError(t, err)
	gzf, err := gzip.NewReader(tf)
	require.NoError(t, err)
	tr := tar.NewReader(gzf)

	symlinkEntriesFound := make(map[string]string) // map of symlink name to link target
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Typeflag == tar.TypeSymlink {
			symlinkEntriesFound[header.Name] = header.Linkname
		}
	}
	_ = gzf.Close()
	_ = tf.Close()

	// Verify both symlinks were stored as symlink entries in the tarball
	require.Equal(t, "target.txt", symlinkEntriesFound["link_to_target.txt"],
		"tarball should contain symlink entry for link_to_target.txt pointing to target.txt")
	require.Equal(t, "../target.txt", symlinkEntriesFound["subdir/link_to_parent.txt"],
		"tarball should contain symlink entry for subdir/link_to_parent.txt pointing to ../target.txt")

	// Extract to new directory
	extractDir := testcommon.CreateTmpDir(t.Name() + "_extract")
	t.Cleanup(func() {
		_ = os.RemoveAll(extractDir)
	})

	err = archive.Untar(tarballFile.Name(), extractDir, "")
	require.NoError(t, err)

	// Verify the regular file exists
	extractedFile := filepath.Join(extractDir, "target.txt")
	assert.FileExists(extractedFile)

	// Verify the symlink in root exists and points to correct target
	extractedSymlink := filepath.Join(extractDir, "link_to_target.txt")
	linkInfo, err := os.Lstat(extractedSymlink)
	require.NoError(t, err)
	assert.True(linkInfo.Mode()&os.ModeSymlink != 0, "link_to_target.txt should be a symlink")

	linkTarget, err := os.Readlink(extractedSymlink)
	require.NoError(t, err)
	assert.Equal("target.txt", linkTarget)

	// Verify we can read through the symlink
	content, err := os.ReadFile(extractedSymlink)
	require.NoError(t, err)
	assert.Equal("test content", string(content))

	// Verify the symlink in subdir exists and points to correct target
	extractedSymlinkInSubdir := filepath.Join(extractDir, "subdir", "link_to_parent.txt")
	linkInfo2, err := os.Lstat(extractedSymlinkInSubdir)
	require.NoError(t, err)
	assert.True(linkInfo2.Mode()&os.ModeSymlink != 0, "subdir/link_to_parent.txt should be a symlink")

	linkTarget2, err := os.Readlink(extractedSymlinkInSubdir)
	require.NoError(t, err)
	linkTarget2 = filepath.ToSlash(linkTarget2)
	assert.Equal("../target.txt", linkTarget2)
}
