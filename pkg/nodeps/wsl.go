package nodeps

import (
	"os"
	"runtime"
	"strings"
)

// IsWSL2 returns true if running WSL2
func IsWSL2() bool {
	if runtime.GOOS == "linux" {
		// First, try checking env variable
		if os.Getenv(`WSL_INTEROP`) != "" {
			return true
		}
		// But that doesn't always work, so check for existence of micorosft in /proc/version
		fullFileBytes, err := os.ReadFile("/proc/version")
		if err != nil {
			return false
		}
		fullFileString := string(fullFileBytes)
		return strings.Contains(fullFileString, "-microsoft")
	}
	return false
}
