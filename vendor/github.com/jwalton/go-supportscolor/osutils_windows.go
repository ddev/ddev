// +build windows

package supportscolor

import (
	"golang.org/x/sys/windows"
)

func getWindowsVersion() (majorVersion, minorVersion, buildNumber uint32) {
	return windows.RtlGetNtVersionNumbers()
}

func enableColor() bool {
	handle, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return false
	}

	// Get the existing console mode.
	var mode uint32
	err = windows.GetConsoleMode(handle, &mode)
	if err != nil {
		return false
	}

	// If ENABLE_VIRTUAL_TERMINAL_PROCESSING is not set, then set it.  This will
	// enable native ANSI color support from Windows.
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING {
		// Enable color.
		// See https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences.
		mode = mode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		err = windows.SetConsoleMode(handle, mode)
		if err != nil {
			return false
		}
	}

	return true
}
