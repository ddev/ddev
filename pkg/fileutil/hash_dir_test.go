package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/stretchr/testify/require"
)

func TestHashDir(t *testing.T) {
	dir := t.TempDir()

	// Create some test files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "subdir", "file2.txt"), []byte("world"), 0644))

	hash1, err := fileutil.HashDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, hash1)

	// Same content should produce same hash
	hash2, err := fileutil.HashDir(dir)
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)

	// Modifying a file should change the hash
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("changed"), 0644))
	hash3, err := fileutil.HashDir(dir)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash3)

	// Non-existent directory should return error
	_, err = fileutil.HashDir(filepath.Join(dir, "nonexistent"))
	require.Error(t, err)
}

func TestHashDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir1, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "b.txt"), []byte("bbb"), 0644))

	hash1, err := fileutil.HashDirs([]string{dir1, dir2})
	require.NoError(t, err)
	require.NotEmpty(t, hash1)

	// Changing extraStrings should change the hash
	hash2, err := fileutil.HashDirs([]string{dir1, dir2}, "image:v1")
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash2)

	// Non-existent directory should be handled gracefully
	hash3, err := fileutil.HashDirs([]string{dir1, filepath.Join(dir1, "missing")})
	require.NoError(t, err)
	require.NotEmpty(t, hash3)
}
