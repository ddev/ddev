//go:build windows

package cmd

import (
	"os/exec"
	"strconv"
)

// setProcessGroupAttr is a no-op on Windows. taskkill with /T traverses the
// Windows job/process tree natively, so no special attribute is required.
func setProcessGroupAttr(_ *exec.Cmd) {
}

// killProcessTree uses taskkill to terminate cmd and all of its descendants.
// /T walks the process tree, /F forces termination of processes that ignore
// the signal. Falls back to a direct Process.Kill if taskkill itself fails.
func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// /T kills the process tree, /F forces termination
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
			_ = cmd.Process.Kill()
		}
	}
}
