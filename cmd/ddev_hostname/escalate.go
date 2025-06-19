package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func escalateIfNeeded() {
	// If we’re not root (UID 0), re‐exec via sudo
	if syscall.Geteuid() != 0 {
		// Prepend our own path to the args
		args := append([]string{os.Args[0]}, os.Args[1:]...)
		cmd := exec.Command("sudo", args...)
		// Pass through the terminal’s stdin/stdout/stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to escalate: %v\n", err)
			os.Exit(1)
		}
		// If sudo succeeds, it will have done the real work,
		// so we just exit in the parent process.
		os.Exit(0)
	}
	// else: we’re already root, continue
}
