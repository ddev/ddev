package nodeps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ddev/ddev/pkg/output"
)

// IsWSL2 returns true if running WSL2
func IsWSL2() bool {
	if !IsLinux() {
		return false
	}
	// First, try checking env variable
	if os.Getenv(`WSL_INTEROP`) != "" {
		return true
	}
	// But that doesn't always work, so check for existence of microsoft in /proc/version
	fullFileBytes, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	fullFileString := string(fullFileBytes)
	return strings.Contains(fullFileString, "-microsoft")
}

// IsWSL2MirroredMode returns true if running WSL2 in mirrored mode.
func IsWSL2MirroredMode() bool {
	if !IsWSL2() {
		return false
	}
	mode, err := GetWSL2NetworkingMode()
	if err != nil {
		output.UserErr.Warnf("Unable to get WSL2 networking mode: %v", err)
		return false
	}
	return mode == "mirrored"
}

// GetWSL2NetworkingMode returns the current WSL2 networking mode,
// normally either "nat" or "mirrored".
func GetWSL2NetworkingMode() (string, error) {
	out, err := exec.Command("wslinfo", "--networking-mode").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run wslinfo: %w", err)
	}
	mode := strings.TrimSpace(strings.ToLower(string(bytes.TrimSpace(out))))
	if mode != "nat" && mode != "mirrored" {
		return "", fmt.Errorf("unrecognized networking mode %q", mode)
	}
	return mode, nil
}

// IsPathOnWindowsFilesystem checks if the given path is on the Windows filesystem
// when running in WSL2. The Windows filesystem is mounted under /mnt/ in WSL2.
func IsPathOnWindowsFilesystem(path string) bool {
	return strings.HasPrefix(path, "/mnt/")
}
