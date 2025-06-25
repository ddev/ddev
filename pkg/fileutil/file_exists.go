//go:build !windows

package fileutil

import "os"

// FileExists checks a file's existence
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
