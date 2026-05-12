//go:build windows

package dockerutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDupStdinWindowsPassThrough verifies that on Windows dupStdin returns
// the SAME *os.File it was given. Returning a different file would break
// the asymmetry SetExecStdin's restore path depends on: compose's
// restoreTerminal explicitly skips the in.Close() on Windows, so a fresh
// dup would leak instead of being closed by the upstream layer.
func TestDupStdinWindowsPassThrough(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	dup, err := dupStdin(r)
	require.NoError(t, err)
	require.NotNil(t, dup)

	// Pass-through contract: same fd as input.
	require.Equal(t, r.Fd(), dup.Fd(),
		"dupStdin must be a pass-through on Windows; SetExecStdin restore depends on this")
	// Same *os.File pointer — SetExecStdin's `dup != f` filter relies on this.
	require.Same(t, r, dup,
		"dupStdin must return the exact same *os.File on Windows")
}
