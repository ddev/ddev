package fileutil

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// FileExists checks a file's existence
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		if errors.Is(err, syscall.Errno(windows.ERROR_INVALID_NAME)) {
			return false
		}
	}
	return true
}
