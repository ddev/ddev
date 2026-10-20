package ddevapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMutagenHostRoot tests which host path is handed to Mutagen as the sync root
func TestMutagenHostRoot(t *testing.T) {
	// EvalSymlinks the temp dir so a symlinked TMPDIR, such as macOS's /var, doesn't affect comparisons
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	realRoot := filepath.Join(tmpDir, "real")
	require.NoError(t, os.Mkdir(realRoot, 0o755))

	linkRoot := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("unable to create symlink: %v", err)
	}

	require.Equal(t, realRoot, mutagenHostRoot(realRoot))
	require.Equal(t, realRoot, mutagenHostRoot(linkRoot))

	// Only a symlinked final component is resolved, because Mutagen accepts symlinks above the root
	nested := filepath.Join(linkRoot, "nested")
	require.NoError(t, os.Mkdir(nested, 0o755))
	require.Equal(t, nested, mutagenHostRoot(nested))

	missing := filepath.Join(tmpDir, "missing")
	require.Equal(t, missing, mutagenHostRoot(missing))
}
