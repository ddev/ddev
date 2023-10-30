//go:build !windows
// +build !windows

package fileutil

import (
	"os"
	"regexp"
)

// Replace SHELL variables with real value
func replaceShellVariables(input string) string {
	r := regexp.MustCompile(`\$(\w+)|\$\{(\w+)\}`)
	return r.ReplaceAllStringFunc(input, func(s string) string {
		varName := r.ReplaceAllString(s, "$1$2")
		return os.Getenv(varName)
	})
}

// FileExists checks a file's existence
func FileExists(name string) bool {
	expandedName := replaceShellVariables(name)
	if _, err := os.Stat(expandedName); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
