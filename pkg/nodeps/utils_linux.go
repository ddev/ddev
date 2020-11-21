package nodeps

import (
	"os"
)

// GetWSLDistro returns the WSL distro name on Linux in WSL
func GetWSLDistro() string {
	return os.Getenv("WSL_DISTRO_NAME")
}

// IsWSL returns true if running in WSL
func IsWSL() bool {
	return GetWSLDistro() != ""
}
