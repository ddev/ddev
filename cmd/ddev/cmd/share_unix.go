//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
	"time"
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

// stopShareProvider gives the provider time to release any remote state.
func stopShareProvider(cmd *exec.Cmd, done <-chan error, grace time.Duration) error {
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		killProcessTree(cmd)
		return <-done
	}

	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case err := <-done:
		killProcessTree(cmd)
		return err
	case <-timer.C:
		killProcessTree(cmd)
		return <-done
	}
}
