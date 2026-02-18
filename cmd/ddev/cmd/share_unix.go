//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

// setProcessGroupAttr places the child process in its own process group so that
// killProcessTree can address the entire group by negating the PID.
func setProcessGroupAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree sends SIGKILL to the process group of cmd. Using a negative
// PID with syscall.Kill targets every process in the group, ensuring child
// processes spawned by the share provider are also terminated.
func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
