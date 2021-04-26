// Package hasflag checks if `os.Args` has a specific flag.  Correctly stops looking
// after an -- argument terminator.
//
// Ported from https://github.com/sindresorhus/has-flag
//
package hasflag

import (
	"os"
	"strings"
)

func argIndexOf(argv []string, str string) int {
	result := -1
	for index, arg := range argv {
		if arg == str {
			result = index
			break
		}
	}

	return result
}

// HasFlag checks to see if the given flag was supplied on the command line.
//
func HasFlag(flag string) bool {
	return hasFlag(flag, os.Args[1:])
}

func hasFlag(flag string, argv []string) bool {
	var prefix string

	if strings.HasPrefix(flag, "-") {
		prefix = ""
	} else if len(flag) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	position := argIndexOf(argv, prefix+flag)
	terminatorPosition := argIndexOf(argv, "--")
	return position != -1 && (terminatorPosition == -1 || position < terminatorPosition)
}
