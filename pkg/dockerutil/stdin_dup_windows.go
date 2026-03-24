//go:build windows

package dockerutil

import "os"

// dupStdin returns stdin unchanged on Windows because restoreTerminal does not
// call in.Close() on Windows (see hijack.go), so no dup is needed.
func dupStdin(stdin *os.File) (*os.File, error) {
	return stdin, nil
}
