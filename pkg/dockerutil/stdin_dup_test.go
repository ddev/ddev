//go:build !windows

package dockerutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDupStdinPOSIX verifies that on POSIX systems dupStdin returns a *os.File
// backed by a NEW file descriptor: closing the returned file must not affect
// the caller's original *os.File. Without this guarantee, compose's hijack
// restoreTerminal would close ddev's real fd 0 on TTY exec exit.
func TestDupStdinPOSIX(t *testing.T) {
	// Use a real *os.File backed by os.Pipe() so the test does not depend on
	// whatever the test harness wired stdin to.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })

	dup, err := dupStdin(r)
	require.NoError(t, err)
	require.NotNil(t, dup)

	// POSIX contract: dup must use a different fd from the source.
	require.NotEqual(t, r.Fd(), dup.Fd(),
		"dupStdin must return a fresh fd on POSIX so closing the dup does not steal the caller's fd")

	// Closing the dup once must succeed.
	require.NoError(t, dup.Close())

	// A second close on the same *os.File must not panic. The underlying fd
	// is already closed, so an error is fine; what we forbid is a crash.
	require.NotPanics(t, func() { _ = dup.Close() })

	// The original *os.File must still be usable after the dup was closed:
	// this proves we did not accidentally close the caller's fd.
	require.NoError(t, r.Close())
}
