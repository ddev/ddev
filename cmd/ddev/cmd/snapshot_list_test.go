package cmd

import (
	"bytes"
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/stretchr/testify/require"
)

// captureListSnapshots runs listSnapshots(apps) with output.UserOut
// redirected to a buffer, restoring the original output when done, and
// returns what would otherwise have gone to stdout.
func captureListSnapshots(t *testing.T, apps []*ddevapp.DdevApp) string {
	t.Helper()
	var buf bytes.Buffer
	orig := output.UserOut.Out
	output.UserOut.SetOutput(&buf)
	t.Cleanup(func() {
		output.UserOut.SetOutput(orig)
	})
	listSnapshots(apps)
	return buf.String()
}

// TestListSnapshotsNoSnapshots verifies `ddev snapshot --list` reports "No
// snapshots" for a project with an empty (or absent) db_snapshots directory,
// and doesn't show a Worktree column when the project isn't in a git worktree
// with sibling snapshots.
func TestListSnapshotsNoSnapshots(t *testing.T) {
	app := &ddevapp.DdevApp{AppRoot: t.TempDir(), Name: "empty-project"}

	out := captureListSnapshots(t, []*ddevapp.DdevApp{app})
	require.Contains(t, out, "No snapshots")
	require.NotContains(t, out, "WORKTREE")
}

// TestListSnapshotsWorktrees verifies `ddev snapshot --list` includes
// snapshots found in a sibling git worktree of the same project alongside the
// project's own, labels each row's source worktree, and leaves the Worktree
// column out entirely for a project with no sibling snapshots to show.
func TestListSnapshotsWorktrees(t *testing.T) {
	tmpDir := t.TempDir()
	resolvedTmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	tmpDir = resolvedTmpDir

	repoRoot := filepath.Join(tmpDir, "repo")
	worktreeRoot := filepath.Join(tmpDir, "repo-worktree")
	projectRoot := filepath.Join(repoRoot, "web")
	projectWorktreeRoot := filepath.Join(worktreeRoot, "web")

	runGit := func(dir string, args ...string) {
		out, err := osexec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
	}

	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	runGit(tmpDir, "init", repoRoot)
	runGit(repoRoot, "config", "user.name", "Test User")
	runGit(repoRoot, "config", "user.email", "test@example.com")
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("placeholder"), 0o644))
	runGit(repoRoot, "add", ".")
	runGit(repoRoot, "commit", "-m", "initial")
	runGit(repoRoot, "worktree", "add", "--detach", worktreeRoot, "HEAD")

	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, ".ddev", "db_snapshots"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectWorktreeRoot, ".ddev", "db_snapshots"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".ddev", "db_snapshots", "current-mariadb_10.11.zst"), []byte("current"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectWorktreeRoot, ".ddev", "db_snapshots", "other-mariadb_10.11.zst"), []byte("other"), 0o644))

	app := &ddevapp.DdevApp{AppRoot: projectRoot, Name: "web"}
	out := captureListSnapshots(t, []*ddevapp.DdevApp{app})

	require.Contains(t, out, "WORKTREE", "a sibling snapshot exists, so the Worktree column should be shown")
	require.Contains(t, out, "current")
	require.Contains(t, out, "other")
	require.Contains(t, out, "web", "the sibling snapshot's row should be labeled with its source worktree directory")
	require.Contains(t, out, "mariadb_10.11")

	// The same project on its own, with no sibling worktree snapshots, should
	// not show a Worktree column at all.
	worktreeApp := &ddevapp.DdevApp{AppRoot: projectWorktreeRoot, Name: "web-worktree"}
	require.NoError(t, os.RemoveAll(filepath.Join(projectRoot, ".ddev", "db_snapshots", "current-mariadb_10.11.zst")))
	soloOut := captureListSnapshots(t, []*ddevapp.DdevApp{worktreeApp})
	require.NotContains(t, soloOut, "WORKTREE")
	require.Contains(t, soloOut, "other")
}
