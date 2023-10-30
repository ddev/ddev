//go:build !windows
// +build !windows

package fileutil

import (
	"os"
	"regexp"
)

func replaceBashVariables(input string) string {
	r := regexp.MustCompile(`\$(\w+)|\$\{(\w+)\}`)
	return r.ReplaceAllStringFunc(input, func(s string) string {
		varName := r.ReplaceAllString(s, "$1$2")
		return os.Getenv(varName)
	})
}

// FileExists checks a file's existence
func FileExists(name string) bool {
	expandedName := replaceBashVariables(name)
	if _, err := os.Stat(expandedName); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
