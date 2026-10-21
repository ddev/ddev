package ddevapp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestGetDBVersionFromVolumeScript verifies the shell script shared by both
// getDBVersionFromVolume() code paths - the utility-container mount case and
// the exec-into-the-running-db-container case added for #7372 - against
// arbitrary file locations, run locally with no Docker involved. It exercises
// the script's own logic (check order, glob handling, "nothing found" case)
// independent of which literal paths either caller passes in.
func TestGetDBVersionFromVolumeScript(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, root string)
		expectedOutput string
	}{
		{
			name: "MariaDB version file present",
			setup: func(t *testing.T, root string) {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "mysql"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "mysql", "db_mariadb_version.txt"), []byte("mariadb_10.11"), 0644))
			},
			expectedOutput: "mariadb_10.11",
		},
		{
			name: "PostgreSQL pre-18 flat PG_VERSION",
			setup: func(t *testing.T, root string) {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "postgres"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "postgres", "PG_VERSION"), []byte("17"), 0644))
			},
			expectedOutput: "17",
		},
		{
			name: "PostgreSQL 18+ version-specific directory",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "postgres", "18", "docker")
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "PG_VERSION"), []byte("18"), 0644))
			},
			expectedOutput: "18",
		},
		{
			name: "PostgreSQL future version-specific directory",
			setup: func(t *testing.T, root string) {
				dir := filepath.Join(root, "postgres", "19", "docker")
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "PG_VERSION"), []byte("19"), 0644))
			},
			expectedOutput: "19",
		},
		{
			name: "MariaDB file takes precedence when both exist",
			setup: func(t *testing.T, root string) {
				require.NoError(t, os.MkdirAll(filepath.Join(root, "mysql"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "mysql", "db_mariadb_version.txt"), []byte("mariadb_10.11"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(root, "postgres"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(root, "postgres", "PG_VERSION"), []byte("17"), 0644))
			},
			expectedOutput: "mariadb_10.11",
		},
		{
			name:           "no version files found",
			setup:          func(t *testing.T, root string) {},
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			// The script runs under bash even on Windows (Git Bash), where an
			// unquoted backslash is an escape character, so paths fed into it
			// must use forward slashes rather than the native os.PathSeparator.
			script := getDBVersionFromVolumeScript(
				filepath.ToSlash(filepath.Join(root, "mysql", "db_mariadb_version.txt")),
				filepath.ToSlash(filepath.Join(root, "postgres", "PG_VERSION")),
				filepath.ToSlash(filepath.Join(root, "postgres", "*", "docker", "PG_VERSION")),
			)
			out, err := exec.Command("bash", "-c", script).CombinedOutput()
			require.NoError(t, err, "script output: %s", out)
			require.Equal(t, tt.expectedOutput, strings.TrimSpace(string(out)))
		})
	}
}

// TestCreateDumpFileDevPath checks that a /dev/fd path backed by a regular
// file, as with `ddev export-db -f /dev/stdout >> backup.sql`, is appended to.
func TestCreateDumpFileDevPath(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("/dev/fd does not exist on Windows")
	}
	out := filepath.Join(t.TempDir(), "backup.sql")
	require.NoError(t, os.WriteFile(out, []byte("earlier\n"), 0644))
	redirected, err := os.OpenFile(out, os.O_WRONLY|os.O_APPEND, 0)
	require.NoError(t, err)
	defer redirected.Close()

	f, finish, err := createDumpFile(fmt.Sprintf("/dev/fd/%d", redirected.Fd()))
	require.NoError(t, err)
	_, err = f.WriteString("dump\n")
	require.NoError(t, err)
	require.NoError(t, finish(true))

	content, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "earlier\ndump\n", string(content))
	entries, err := os.ReadDir(filepath.Dir(out))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

// TestCreateDumpFileUnderDev checks that a regular file elsewhere under /dev
// is created or replaced like any other dump file.
func TestCreateDumpFileUnderDev(t *testing.T) {
	if !fileutil.IsDirectory("/dev/shm") {
		t.Skip("/dev/shm is not available")
	}
	out, err := os.MkdirTemp("/dev/shm", "ddev-test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(out) })
	out = filepath.Join(out, "db.sql")

	for _, want := range []string{"first\n", "second\n"} {
		f, finish, err := createDumpFile(out)
		require.NoError(t, err)
		_, err = f.WriteString(want)
		require.NoError(t, err)
		require.NoError(t, finish(true))
		content, err := os.ReadFile(out)
		require.NoError(t, err)
		require.Equal(t, want, string(content))
	}
}
