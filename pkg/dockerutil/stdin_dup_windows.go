//go:build windows

package dockerutil

import "os"

// dupStdin is a pass-through on Windows: docker/cli's restoreTerminal explicitly
// skips the in.Close() on windows (and darwin) to avoid blocking on
// CloseHandle, so the real fd 0 is never at risk and a dup buys nothing.
// See restoreTerminal in
// vendor/github.com/docker/cli/cli/command/container/hijack.go (line 211)
// (https://github.com/docker/cli/blob/v29.4.0/cli/command/container/hijack.go#L211)
// where the GOOS guard is enforced.
//
// Returning the original *os.File means the caller's restore path must NOT
// close it; the non-windows implementation returns a fresh fd that the caller
// owns. dupCloser in docker_compose.go handles that asymmetry.
func dupStdin(stdin *os.File) (*os.File, error) {
	return stdin, nil
}
