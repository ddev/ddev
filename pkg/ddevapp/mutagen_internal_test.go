package ddevapp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
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

	relLink := filepath.Join(tmpDir, "rel-link")
	require.NoError(t, os.Symlink("real", relLink))
	require.Equal(t, realRoot, mutagenHostRoot(relLink))

	chained := filepath.Join(tmpDir, "chained")
	require.NoError(t, os.Symlink(linkRoot, chained))
	require.Equal(t, realRoot, mutagenHostRoot(chained))

	cycleA := filepath.Join(tmpDir, "cycle-a")
	cycleB := filepath.Join(tmpDir, "cycle-b")
	require.NoError(t, os.Symlink(cycleB, cycleA))
	require.NoError(t, os.Symlink(cycleA, cycleB))
	require.Equal(t, cycleA, mutagenHostRoot(cycleA))
}

// TestSessionRootMatches tests when an existing sync session's host path is treated as stale
func TestSessionRootMatches(t *testing.T) {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	realA := filepath.Join(tmpDir, "real-a")
	realB := filepath.Join(tmpDir, "real-b")
	require.NoError(t, os.Mkdir(realA, 0o755))
	require.NoError(t, os.Mkdir(realB, 0o755))
	linkRoot := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realA, linkRoot); err != nil {
		t.Skipf("unable to create symlink: %v", err)
	}

	require.True(t, sessionRootMatches(realA, realA))
	require.True(t, sessionRootMatches("", realA))
	require.True(t, sessionRootMatches(realA, linkRoot))
	// A session created from the symlink itself, which Mutagen can't sync
	require.False(t, sessionRootMatches(linkRoot, linkRoot))

	require.NoError(t, os.Remove(linkRoot))
	require.NoError(t, os.Symlink(realB, linkRoot))
	require.False(t, sessionRootMatches(realA, linkRoot))

	// The old target is gone
	require.NoError(t, os.Remove(realA))
	require.False(t, sessionRootMatches(realA, linkRoot))

	// The project directory itself is missing, such as an unmounted drive
	require.NoError(t, os.Remove(realB))
	require.True(t, sessionRootMatches(filepath.Join(tmpDir, "elsewhere"), linkRoot))
}

// TestMutagenHostRootJunction tests that a Windows junction is resolved like a symlink
func TestMutagenHostRootJunction(t *testing.T) {
	if !nodeps.IsWindows() {
		t.Skip("junctions exist only on Windows")
	}
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	realRoot := filepath.Join(tmpDir, "real")
	require.NoError(t, os.Mkdir(realRoot, 0o755))
	junction := filepath.Join(tmpDir, "junction")
	// mklink /J needs no Developer Mode or admin rights, unlike os.Symlink
	out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, realRoot).CombinedOutput()
	require.NoError(t, err, "mklink /J failed: %s", out)

	require.True(t, isLinkLeaf(junction))
	require.False(t, isLinkLeaf(realRoot))
	require.Equal(t, realRoot, mutagenHostRoot(junction))
	require.True(t, sessionRootMatches(realRoot, junction))
	require.False(t, sessionRootMatches(junction, junction))
}
