package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

func TestLaunchGUIApp(t *testing.T) {
	if nodeps.IsWindows() {
		t.Skip("the stand-in apps are shell scripts")
	}
	dir := t.TempDir()
	app := func(name, body string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0755))
		return p
	}

	err := launchGUIApp(app("fails", `echo "libwebkitgtk-6.0.so.4: cannot open shared object file" >&2; exit 127`), "")
	require.ErrorContains(t, err, "exit status 127")
	require.ErrorContains(t, err, "cannot open shared object file")

	// A second launch passes its arguments to the running app and exits.
	require.NoError(t, launchGUIApp(app("hands-off", "exit 0"), "my-project"))
	require.NoError(t, launchGUIApp(app("runs", "sleep 3"), "my-project"))
}
