package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChooseMode(t *testing.T) {
	t.Setenv("DDEV_DESKTOP_MODE", "loopback")
	m, err := chooseMode()
	require.NoError(t, err)
	require.Equal(t, modeLoopback, m)

	t.Setenv("DDEV_DESKTOP_MODE", "browser")
	_, err = chooseMode()
	require.ErrorContains(t, err, "must be embedded, loopback, or wsl")
}

// TestRemoteServer drives the code path of WSL2 mode against a local
// `ddev serve`. It needs DDEV_BINARY, a ddev built from this branch.
func TestRemoteServer(t *testing.T) {
	bin := os.Getenv("DDEV_BINARY")
	if bin == "" {
		t.Skip("DDEV_BINARY is not set")
	}
	t.Setenv("DDEV_XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DDEV_NO_INSTRUMENTATION", "true")

	r, err := startRemote(bin)
	require.NoError(t, err)
	t.Cleanup(r.Close)
	require.Regexp(t, `^http://127\.0\.0\.1:\d+/\?token=[0-9a-f]{64}$`, r.URL())

	// Without Docker the status carries an error instead of projects. With
	// it, running projects are listed even outside this configuration.
	if _, err := r.Projects(); err != nil {
		require.ErrorContains(t, err, "Docker")
	}

	require.ErrorContains(t, r.Post("/api/projects/nonexistent/actions/start"), "nonexistent")
	require.Error(t, r.Post("/api/actions/nothing"))

	r.Close()
	select {
	case <-r.exited:
	case <-time.After(10 * time.Second):
		t.Fatal("ddev serve did not stop when its stdin closed")
	}
}
