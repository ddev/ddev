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

// IsWSL2HostAddressLoopbackEnabled checks if hostAddressLoopback=true is set
// in the Windows .wslconfig file. This setting is required for WSL2 mirrored
// networking mode to allow containers to connect back to the Windows host.
// Returns true if enabled, false otherwise.
func IsWSL2HostAddressLoopbackEnabled() bool {
	if !IsWSL2() {
		return false
	}

	wslConfigPath := GetWSLConfigPath()
	if wslConfigPath == "" {
		return false
	}

	content, err := os.ReadFile(wslConfigPath)
	if err != nil {
		return false
	}

	return ParseWSLConfigHostAddressLoopback(string(content))
}

// GetWSLConfigPath returns the path to the Windows .wslconfig file
// when running in WSL2, or empty string if not found/accessible.
func GetWSLConfigPath() string {
	if !IsWSL2() {
		return ""
	}

	// Get Windows username from the Windows environment
	// Try multiple approaches to find the user's home directory
	winUserProfile := os.Getenv("USERPROFILE")
	if winUserProfile != "" {
		// Convert Windows path to WSL path: C:\Users\foo -> /mnt/c/Users/foo
		winUserProfile = strings.ReplaceAll(winUserProfile, "\\", "/")
		if len(winUserProfile) >= 2 && winUserProfile[1] == ':' {
			drive := strings.ToLower(string(winUserProfile[0]))
			wslPath := "/mnt/" + drive + winUserProfile[2:]
			configPath := wslPath + "/.wslconfig"
			if _, err := os.Stat(configPath); err == nil {
				return configPath
			}
		}
	}

	// Fallback: try to get the Windows username via cmd.exe
	cmd := exec.Command("cmd.exe", "/c", "echo %USERPROFILE%")
	out, err := cmd.Output()
	if err == nil {
		winPath := strings.TrimSpace(string(out))
		winPath = strings.ReplaceAll(winPath, "\\", "/")
		winPath = strings.TrimSuffix(winPath, "\r")
		if len(winPath) >= 2 && winPath[1] == ':' {
			drive := strings.ToLower(string(winPath[0]))
			wslPath := "/mnt/" + drive + winPath[2:]
			configPath := wslPath + "/.wslconfig"
			if _, err := os.Stat(configPath); err == nil {
				return configPath
			}
		}
	}

	return ""
}

// ParseWSLConfigHostAddressLoopback parses .wslconfig content and returns
// true if hostAddressLoopback=true is found under [experimental] section.
func ParseWSLConfigHostAddressLoopback(content string) bool {
	inExperimentalSection := false
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Handle Windows CRLF
		line = strings.TrimSuffix(line, "\r")

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			inExperimentalSection = (sectionName == "experimental")
			continue
		}

		// Look for hostAddressLoopback in [experimental] section
		if inExperimentalSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(strings.ToLower(parts[0]))
				value := strings.TrimSpace(strings.ToLower(parts[1]))
				if key == "hostaddressloopback" && value == "true" {
					return true
				}
			}
		}
	}

	return false
}
