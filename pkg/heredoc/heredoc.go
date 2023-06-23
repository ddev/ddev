package heredoc

import (
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
)

var indentRE = regexp.MustCompile(`(?m)^`)

// Indent returns a copy of the string s with indent prefixed to it, will apply
// indent to each line of the string except the last line if empty.
func Indent(s, indent string) string {
	if len(strings.TrimSpace(s)) == 0 {
		return s
	}

	return strings.TrimSuffix(indentRE.ReplaceAllLiteralString(s, indent), indent)
}

// Doc returns un-indented string as here-document.
func Doc(raw string) string {
	return heredoc.Doc(raw)
}

// DocIndent returns string as here-document indented by indent.
func DocIndent(raw, indent string) string {
	return Indent(
		heredoc.Doc(raw),
		indent,
	)
}

// DocI2S returns string as here-document indented by two spaces.
func DocI2S(raw string) string {
	return Indent(
		heredoc.Doc(raw),
		"  ",
	)
}
