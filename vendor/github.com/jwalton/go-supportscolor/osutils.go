// +build !windows

package supportscolor

func getWindowsVersion() (majorVersion, minorVersion, buildNumber uint32) {
	return 0, 0, 0
}

// enableColor will enable color in the terminal.  Returns true if color was
// enabled, false otherwise.
func enableColor() bool {
	return true
}
