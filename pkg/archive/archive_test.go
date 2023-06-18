package archive_test

import (
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

	tarballFile, err := os.CreateTemp("", t.Name()+"_*.tar.gz")
	assert.NoError(err)

	tarSrc := filepath.Join(origDir, "testdata", t.Name())
	err = os.Chdir(tarSrc)
	assert.NoError(err)

	expectations := map[string]fs.FileMode{}
	for _, f := range []string{".test.sh", "root.txt", filepath.Join("subdir1", "subdir1.txt")} {
		fi, err := os.Stat(f)
		assert.NoError(err)
		expectations[f] = fi.Mode()
	}

	err = archive.Tar(tarSrc, tarballFile.Name(), filepath.Join("subdir1", "subdir2"))
	assert.NoError(err)

	tmpDir := testcommon.CreateTmpDir(t.Name())

	err = os.Chdir(tmpDir)
	assert.NoError(err)
	t.Cleanup(
		func() {
			err := os.Chdir(origDir)
			assert.NoError(err)

			// Could not figure out what causes this not to be removable
			//err = os.Remove(tarballFile.Name())
			//assert.NoError(err)
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)
		})
	err = archive.Untar(tarballFile.Name(), tmpDir, "")
	assert.NoError(err)

	for fileName, mode := range expectations {
		testedFileName, err := filepath.Abs(fileName)
		assert.NoError(err, "fileName err: %v %v", testedFileName, err)
		fi, err := os.Stat(fileName)
		assert.NoError(err)
		require.NotNil(t, fi)
		//desc := fmt.Sprintf("%s: Orig mode=%o, found mode=%o", fileName, mode, fi.Mode())
		//t.Log(desc)
		assert.Equal(mode, fi.Mode(), "expected mode for %s was %o but got %o", fileName, mode, fi.Mode())
	}
	assert.NoFileExists(filepath.Join(tmpDir, "subdir1", "subdir2", "s2.txt"))
}

// TestArchiveTar tests creation of a simple tarball
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
			// Could not figure out what causes this not to be removable
			//err = os.Remove(tarballFile.Name())
			//assert.NoError(err)
			err = os.RemoveAll(tmpDir)
			assert.NoError(err)
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
	assert := asrt.New(t)

	testTarball := "https://github.com/ddev/ddev-drupal9-solr/archive/refs/tags/v0.1.1.tar.gz"

	dir, cleanup, err := archive.DownloadAndExtractTarball(testTarball, true)
	require.NoError(t, err)
	assert.DirExists(dir)
	assert.FileExists(path.Join(dir, "install.yaml"))
	cleanup()
	assert.NoDirExists(dir)
}
