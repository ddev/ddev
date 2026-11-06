//go:build !windows

package dockerutil

import (
	"io"
	"os"
	"os/exec"
)

// startWithParent gives the child the read end of a pipe that only ddev
// holds open, so the child sees EOF when ddev exits, even on SIGKILL.
func startWithParent(child *exec.Cmd) (release func(), err error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	child.ExtraFiles = []*os.File{r}
	err = child.Start()
	_ = r.Close()
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	return func() { _ = w.Close() }, nil
}

// exitWithParent runs in the child. It exits once startWithParent's pipe,
// fd 3, closes, so the exec doesn't keep reading the terminal after ddev.
func exitWithParent() {
	restoreTerminal := saveTerminal(os.Stdin)
	go func() {
		_, _ = io.Copy(io.Discard, os.NewFile(3, "parent"))
		restoreTerminal()
		os.Exit(1)
	}()
}
