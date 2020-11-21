// +build !linux

package nodeps

// GetWSLDistro returns the WSL distro name on Linux in WSL
func GetWSLDistro() string {
	return ""
}

// IsWSL returns true if running in WSL
func IsWSL() bool {
	return false
}
