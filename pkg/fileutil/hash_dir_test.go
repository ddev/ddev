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
