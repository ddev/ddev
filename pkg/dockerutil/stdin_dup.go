//go:build !windows

package dockerutil

import (
	"os"
	"syscall"
)

// dupStdin duplicates the stdin file descriptor so DDEV can hand the duplicate
// to docker-compose's exec/run hijack code without risking the real fd 0.
//
// Background: when a TTY exec ends, docker/cli's hijack restore path calls
// in.Close() on the stream it was given, but only on non-darwin / non-windows
// platforms. See restoreTerminal in
// vendor/github.com/docker/cli/cli/command/container/hijack.go (line 211)
// (https://github.com/docker/cli/blob/v29.4.0/cli/command/container/hijack.go#L211).
// The close is wired in by setupInput in the same file (line 95), and only
// when h.tty == true. Without a dup, that close lands on the ddev process's
// real fd 0 — harmless for a one-shot exec that exits immediately, but it
// would break any later in-process read of stdin (e.g. a second TTY exec or
// an interactive prompt during cleanup).
//
// The returned *os.File MUST be closed by the caller's restore path so the
// dup does not leak an fd on platforms where compose's restoreTerminal
// declines to close it (darwin, windows) or on non-TTY paths (where
// hijack's setupInput returns a no-op restore and never calls Close).
// Closing an *os.File whose underlying fd compose already closed is a
// well-defined no-op error and safe to ignore.
func dupStdin(stdin *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(stdin.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "stdin"), nil
}
