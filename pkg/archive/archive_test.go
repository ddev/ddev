package archive_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
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

// TestUntarPathTraversal verifies that path traversal attempts in tar archives are rejected
func TestUntarPathTraversal(t *testing.T) {
	destDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() { _ = os.RemoveAll(destDir) })

	buildTar := func(entryName string, linkname string, typeflag byte) string {
		f, err := os.CreateTemp("", t.Name()+"_*.tar.gz")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Remove(f.Name()) })

		gw := gzip.NewWriter(f)
		tw := tar.NewWriter(gw)

		hdr := &tar.Header{
			Name:     entryName,
			Typeflag: typeflag,
			Linkname: linkname,
			Mode:     0644,
			Size:     0,
		}
		if typeflag == tar.TypeReg {
			hdr.Size = 5
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if typeflag == tar.TypeReg {
			_, err = tw.Write([]byte("hello"))
			require.NoError(t, err)
		}
		require.NoError(t, tw.Close())
		require.NoError(t, gw.Close())
		require.NoError(t, f.Close())
		return f.Name()
	}

	t.Run("traversal_in_file_path", func(t *testing.T) {
		tarball := buildTar("../../traversal_file.txt", "", tar.TypeReg)
		err := archive.Untar(tarball, destDir, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "escapes destination directory")
	})

	t.Run("traversal_in_symlink_target", func(t *testing.T) {
		tarball := buildTar("link.txt", "../../outside.txt", tar.TypeSymlink)
		err := archive.Untar(tarball, destDir, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "escapes destination directory")
	})

	t.Run("absolute_symlink_target_with_traversal", func(t *testing.T) {
		tarball := buildTar("link.txt", "/../../../etc/passwd", tar.TypeSymlink)
		err := archive.Untar(tarball, destDir, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "is absolute, which is not allowed")
	})

	t.Run("absolute_symlink_target_rejected", func(t *testing.T) {
		// Absolute symlink targets are rejected outright, even ones shaped
		// like container paths (e.g. /var/www/html/...). There is no safe
		// general way to let one through: rebasing it under dest is what
		// caused this function's history of escape bugs, see GHSA-9hq4-hm3j-jmph.
		tarball := buildTar("link.txt", "/var/www/html/lib/web/underscore.js", tar.TypeSymlink)
		err := archive.Untar(tarball, destDir, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "is absolute, which is not allowed")
	})

	t.Run("absolute_symlink_write_through_escape", func(t *testing.T) {
		// Regression test for GHSA-9hq4-hm3j-jmph: an absolute symlink target
		// must be rejected outright, since a follow-up regular-file entry
		// that traverses it could otherwise write outside dest.
		victimDir := t.TempDir()
		victimFile := filepath.Join(victimDir, "pwned.txt")
		require.NoError(t, os.WriteFile(victimFile, []byte("original"), 0644))

		f, err := os.CreateTemp("", "absolute_symlink_write_through_escape_*.tar")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.Remove(f.Name()) })

		tw := tar.NewWriter(f)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     "link",
			Typeflag: tar.TypeSymlink,
			Linkname: victimDir,
			Mode:     0777,
		}))
		content := []byte("OWNED-BY-ARCHIVE")
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     "link/pwned.txt",
			Typeflag: tar.TypeReg,
			Size:     int64(len(content)),
			Mode:     0644,
		}))
		_, err = tw.Write(content)
		require.NoError(t, err)
		require.NoError(t, tw.Close())
		require.NoError(t, f.Close())

		writeThroughDest := testcommon.CreateTmpDir("absolute_symlink_write_through_escape")
		t.Cleanup(func() { _ = os.RemoveAll(writeThroughDest) })
		err = archive.Untar(f.Name(), writeThroughDest, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "is absolute, which is not allowed")

		data, err := os.ReadFile(victimFile)
		require.NoError(t, err)
		require.Equal(t, "original", string(data),
			"host file outside dest must not be overwritten via a planted symlink")
	})
}

// TestUnzipPathTraversal verifies that path traversal attempts in zip archives are rejected
func TestUnzipPathTraversal(t *testing.T) {
	destDir := testcommon.CreateTmpDir(t.Name())
	t.Cleanup(func() { _ = os.RemoveAll(destDir) })

	// Build a zip with a traversal entry
	zipFile, err := os.CreateTemp("", t.Name()+"_*.zip")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(zipFile.Name()) })

	zw := zip.NewWriter(zipFile)
	w, err := zw.Create("../../traversal_file.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("pwned"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, zipFile.Close())

	err = archive.Unzip(zipFile.Name(), destDir, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes destination directory")
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

// makeTestArchive writes members to a tar and a zip file in the same
// directory and returns their paths.
func makeTestArchive(t *testing.T, members map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()

	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, content := range members {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(content)), Typeflag: tar.TypeReg}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	tarPath := filepath.Join(dir, "test.tar")
	require.NoError(t, os.WriteFile(tarPath, tarBuf.Bytes(), 0644))

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	for name, content := range members {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	zipPath := filepath.Join(dir, "test.zip")
	require.NoError(t, os.WriteFile(zipPath, zipBuf.Bytes(), 0644))

	return tarPath, zipPath
}

// TestSQLMembers tests that only top-level .sql/.mysql archive members are
// listed, mirroring ImportDB's extraction-then-glob behavior, and that the
// named members can be streamed back in order.
func TestSQLMembers(t *testing.T) {
	members := map[string]string{
		"b.sql":        "SELECT 1;\n",
		"a.sql":        "SELECT 2;\n",
		"nested/c.sql": "SELECT 3;\n",
		"x.sql.gz":     "garbage",
		"notes.txt":    "not sql",
	}
	tarPath, zipPath := makeTestArchive(t, members)

	tests := []struct {
		name          string
		source        string
		extractionDir string
		want          []string
		wantErr       bool
	}{
		{"tar top level", tarPath, "", []string{"a.sql", "b.sql"}, false},
		{"zip top level", zipPath, "", []string{"a.sql", "b.sql"}, false},
		{"tar extraction dir", tarPath, "nested/", []string{"nested/c.sql"}, false},
		{"zip extraction dir", zipPath, "nested/", []string{"nested/c.sql"}, false},
		{"tar extraction dir without slash", tarPath, "nested", []string{"nested/c.sql"}, false},
		{"zip extraction dir without slash", zipPath, "nested", []string{"nested/c.sql"}, false},
		{"tar single file", tarPath, "b.sql", []string{"b.sql"}, false},
		{"zip single file", zipPath, "b.sql", []string{"b.sql"}, false},
		{"tar matching dir without sql", tarPath, "notes.txt", nil, false},
		{"zip matching dir without sql", zipPath, "notes.txt", nil, false},
		{"tar missing extraction dir", tarPath, "missing/", nil, true},
		{"zip missing extraction dir", zipPath, "missing/", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			var err error
			if strings.HasSuffix(tt.source, ".zip") {
				got, err = archive.SQLMembersInZip(tt.source, tt.extractionDir)
			} else {
				got, err = archive.SQLMembersInTar(tt.source, tt.extractionDir)
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	// Streaming concatenates the named members in the given order.
	var buf bytes.Buffer
	require.NoError(t, archive.StreamTarMembers(tarPath, []string{"a.sql", "b.sql"}, &buf))
	require.Equal(t, "SELECT 2;\nSELECT 1;\n", buf.String())
	buf.Reset()
	require.NoError(t, archive.StreamZipMembers(zipPath, []string{"a.sql", "b.sql"}, &buf))
	require.Equal(t, "SELECT 2;\nSELECT 1;\n", buf.String())

	// A gzipped tarball streams the same way.
	dir := t.TempDir()
	gzPath := filepath.Join(dir, "test.tar.gz")
	f, err := os.Create(gzPath)
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	raw, err := os.ReadFile(tarPath)
	require.NoError(t, err)
	_, err = gw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())
	buf.Reset()
	require.NoError(t, archive.StreamTarMembers(gzPath, []string{"a.sql"}, &buf))
	require.Equal(t, "SELECT 2;\n", buf.String())
}

// TestSQLDumpReader tests that plain and gzip-compressed SQL files yield
// their decompressed contents.
func TestSQLDumpReader(t *testing.T) {
	content := "SELECT 1;\n"
	dir := t.TempDir()

	plain := filepath.Join(dir, "db.sql")
	require.NoError(t, os.WriteFile(plain, []byte(content), 0644))
	rc, err := archive.SQLDumpReader(plain)
	require.NoError(t, err)
	b, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, content, string(b))
	require.NoError(t, rc.Close())

	gz := filepath.Join(dir, "db.sql.gz")
	f, err := os.Create(gz)
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	_, err = gw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	rc, err = archive.SQLDumpReader(gz)
	require.NoError(t, err)
	b, err = io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, content, string(b))
	require.NoError(t, rc.Close())

	// A corrupt gzip stream must be reported, not silently truncated.
	bad := filepath.Join(dir, "bad.sql.gz")
	require.NoError(t, os.WriteFile(bad, []byte("not gzip"), 0644))
	_, err = archive.SQLDumpReader(bad)
	require.Error(t, err)
}
