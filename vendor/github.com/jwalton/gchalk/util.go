package gchalk

import (
	"regexp"
	"strings"
)

var crlfRegex = regexp.MustCompile(`(\r\n|\n)`)

func stringEncaseCRLF(str string, prefix string, postfix string) string {
	return crlfRegex.ReplaceAllString(str, prefix+"${1}"+postfix)
}

func strEqualsIgnoreCase(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return strings.EqualFold(a, b)
}
