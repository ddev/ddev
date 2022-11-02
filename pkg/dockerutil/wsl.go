package dockerutil

import (
	"github.com/drud/ddev/pkg/exec"
	"os"
	"runtime"
)

// IsWSL2 returns true if running WSL2
func IsWSL2() bool {
	if runtime.GOOS == "linux" {
		// First, try checking env variable
		if os.Getenv(`WSL_INTEROP`) != "" {
			return true
		}
		// But that doesn't always work, so check for existence of wsl.exe
		_, err := exec.RunHostCommand("command -v wsl.exe")
		if err == nil {
			return true
		}
	}
	return false
}
