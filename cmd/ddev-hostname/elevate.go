//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func elevateIfNeeded() {
	// If we’re not root (UID 0), re‐exec via sudo
	if syscall.Geteuid() == 0 {
		// Already running as root, no need to elevate
		return
	}
	// Prepend our own path to the args
	args := append([]string{os.Args[0]}, os.Args[1:]...)
	cmd := exec.Command("sudo", args...)
	// Pass through the terminal’s stdin/stdout/stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printStderr("Failed to elevate command: %v\n", err)
		os.Exit(1)
	}
	// If sudo succeeds, it will have done the real work,
	// so we just exit in the parent process.
	os.Exit(0)
}
