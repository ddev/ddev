//go:build !windows

package dockerutil

import (
	"os"
	"syscall"
)

// dupStdin duplicates the stdin file descriptor so that when restoreTerminal
// closes it on Linux, only the dup is closed and the original fd 0 remains open.
func dupStdin(stdin *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(stdin.Fd()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "stdin"), nil
}
