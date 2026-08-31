package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/stretchr/testify/require"
)

// TestCmdNestedProjectWarning validates which invocations tell the user they're acting on
// the registered project around an unregistered nested one, including the flags and aliases
// that shouldn't change the answer.
func TestCmdNestedProjectWarning(t *testing.T) {
	testcommon.SkipUnlessDefaultEnvironment(t)
	origDir, _ := os.Getwd()
	parentDir := testcommon.CreateTmpDir(t.Name())
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	t.Cleanup(func() {
		require.NoError(t, os.Chdir(origDir))
		// `ddev config` on the child registers it, so both projects need removing.
		for _, name := range []string{t.Name(), t.Name() + "-child"} {
			out, err := exec.RunHostCommand(DdevBin, "delete", "-Oy", name)
			require.NoError(t, err, "output=%s", out)
		}
		_ = os.RemoveAll(parentDir)
	})

	// Register the outer project, then give the child an unregistered config of its own.
	require.NoError(t, os.Chdir(parentDir))
	out, err := exec.RunHostCommand(DdevBin, "config", "--project-name="+t.Name(), "--omit-containers=db")
	require.NoError(t, err, "output=%s", out)
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, ".ddev"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".ddev", "config.yaml"),
		[]byte("name: "+t.Name()+"-child\ntype: php\nomit_containers: [db]\n"), 0644))
	require.NoError(t, os.Chdir(childDir))

	const warning = "is not registered"

	out, err = exec.RunHostCommand(DdevBin, "describe")
	require.NoError(t, err, "output=%s", out)
	require.Contains(t, out, warning)
	require.Contains(t, out, childDir)

	// JSON output has to stay machine-readable.
	out, err = exec.RunHostCommand(DdevBin, "describe", "-j")
	require.NoError(t, err, "output=%s", out)
	require.NotContains(t, out, warning)

	// A bare `ddev start` asks instead, and DDEV_NONINTERACTIVE (set for these tests) takes
	// the "no" default. A flag on it or the `add` alias is still the bare form.
	for _, args := range [][]string{
		{"start"},
		{"start", "--skip-hooks"},
		{"add"},
	} {
		out, err := exec.RunHostCommand(DdevBin, args...)
		require.NoError(t, err, "output=%s", out)
		require.NotContains(t, out, warning, "`ddev %v` should ask, not warn", args)
		require.Contains(t, out, t.Name(), "`ddev %v` should act on the outer project", args)
	}

	// Naming a project isn't the bare form, and `-y` skips the question; both warn instead.
	for _, args := range [][]string{
		{"start", t.Name()},
		{"start", "-y"},
	} {
		out, err := exec.RunHostCommand(DdevBin, args...)
		require.NoError(t, err, "output=%s", out)
		require.Contains(t, out, warning, "`ddev %v` should warn", args)
	}

	// `ddev config` configures the directory it's in, so the warning would contradict it.
	// This registers the child, so it comes last - afterwards nothing is nested-unregistered.
	out, err = exec.RunHostCommand(DdevBin, "config", "--docroot=.")
	require.NoError(t, err, "output=%s", out)
	require.NotContains(t, out, warning)
	require.Contains(t, out, childDir, "`ddev config` should act on the nested directory")
}
