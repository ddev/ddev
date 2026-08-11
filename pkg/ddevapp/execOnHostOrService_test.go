package ddevapp_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestExecOnHostOrServiceContainerOutput ensures a failing container-service
// command (the branch used by provider examples like acquia.yaml, which
// don't set `service: host`) has its output attached to the returned error
// when stdout is not a terminal, same as the host branch already does.
// Forcing stdout non-tty makes this deterministic regardless of whether the
// test itself runs at an interactive terminal; only stdin's ttyness varies
// by environment, and app.Exec() requires both to skip capture.
func TestExecOnHostOrServiceContainerOutput(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("util.CaptureStdOut() doesn't work on Windows")
	}

	origDir, _ := os.Getwd()
	tmpDir := testcommon.CreateTmpDir(t.Name())

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	app, err := ddevapp.NewApp(tmpDir, true)
	require.NoError(t, err)
	app.Name = t.Name()
	app.Type = nodeps.AppTypePHP
	err = app.WriteConfig()
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		require.NoError(t, err)
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpDir)
	})

	err = app.Start()
	require.NoError(t, err)

	restoreStdout := util.CaptureStdOut()
	err = app.ExecOnHostOrService("web", `echo "boom-output-marker-web" >&2; exit 7`)
	_ = restoreStdout()

	require.Error(t, err)
	require.Contains(t, err.Error(), "boom-output-marker-web")
}
