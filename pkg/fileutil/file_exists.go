//go:build !windows
// +build !windows

package fileutil

import "os"

// FileExists checks a file's existence
func FileExists(name string) bool {
	var finfo os.FileInfo
	var err error
	if finfo, err = os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	_ = finfo
	return true
}
