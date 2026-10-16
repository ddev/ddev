package dotenv

import (
	"maps"
	"regexp"
	"slices"
	"strings"
)

var (
	// Compose also accepts NAME: value. A commented line needs =, so prose
	// like "# Note: ..." is never taken for an assignment.
	assignmentRegex = regexp.MustCompile(`^[ \t]*(export[ \t]+)?([A-Za-z0-9_.\-\[\]]+)[ \t]*[=:][ \t]*`)
	commentedRegex  = regexp.MustCompile(`^[ \t]*#+[ \t]*(export[ \t]+)?([A-Za-z0-9_.\-\[\]]+)[ \t]*=`)
	utf8BOM         = "\xef\xbb\xbf"
)

// assignment is where one NAME=value sits in env file text.
type assignment struct {
	name       string
	export     bool
	commented  bool
	lineStart  int
	valueStart int
	// valueEnd excludes a trailing comment and the line ending.
	valueEnd int
}

// Update returns text with each name in values set to Quote(value).
//
// An existing assignment keeps its place, its `export` prefix and any
// trailing comment, and a quoted value that spans several lines is replaced
// whole. With no assignment, the first commented-out one, like `# DB_HOST=`,
// is uncommented in place. Otherwise the variable is appended, sorted by name,
// matching the file's line endings and its use of `export`.
func Update(text string, values map[string]string) string {
	assignments := findAssignments(text)
	type edit struct {
		start, end  int
		replacement string
	}
	var edits []edit
	var appended []string
	usesExport := slices.ContainsFunc(assignments, func(a assignment) bool { return a.export && !a.commented })

	for _, name := range slices.Sorted(maps.Keys(values)) {
		quoted := Quote(values[name])
		var active []assignment
		var commented *assignment
		for i, a := range assignments {
			switch {
			case a.name != name:
			case !a.commented:
				active = append(active, a)
			case commented == nil:
				commented = &assignments[i]
			}
		}
		switch {
		case len(active) > 0:
			for _, a := range active {
				edits = append(edits, edit{a.valueStart, a.valueEnd, quoted})
			}
		case commented != nil:
			prefix := ""
			if commented.export {
				prefix = "export "
			}
			edits = append(edits, edit{commented.lineStart, commented.valueEnd, prefix + name + "=" + quoted})
		default:
			prefix := ""
			if usesExport {
				prefix = "export "
			}
			appended = append(appended, prefix+name+"="+quoted)
		}
	}

	slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
	for _, e := range edits {
		text = text[:e.start] + e.replacement + text[e.end:]
	}

	if len(appended) > 0 {
		newline := "\n"
		if strings.Contains(text, "\r\n") {
			newline = "\r\n"
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += newline
		}
		text += strings.Join(appended, newline) + newline
	}
	return text
}

// ParseLiteral parses env file text like Parse, but keeps $ references as
// written. It is for an application's own .env, which its framework reads
// with its own expansion rules.
func ParseLiteral(path, text string) (map[string]string, error) {
	var b strings.Builder
	last := 0
	for _, a := range findAssignments(text) {
		value := text[a.valueStart:a.valueEnd]
		if a.commented || strings.HasPrefix(value, "'") {
			continue
		}
		b.WriteString(text[last:a.valueStart])
		b.WriteString(escapeDollars(value, strings.HasPrefix(value, `"`)))
		last = a.valueEnd
	}
	b.WriteString(text[last:])
	return Parse(path, b.String(), nil)
}

// escapeDollars doubles each $ so compose keeps it, except one a backslash
// already escapes inside double quotes.
func escapeDollars(value string, doubleQuoted bool) string {
	var b strings.Builder
	backslashes := 0
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == '$' && !(doubleQuoted && backslashes%2 == 1) {
			b.WriteByte('$')
		}
		if c == '\\' {
			backslashes++
		} else {
			backslashes = 0
		}
		b.WriteByte(c)
	}
	return b.String()
}

// findAssignments locates every assignment in text, commented-out ones
// included, following the quoting rules of the compose parser.
func findAssignments(text string) []assignment {
	var found []assignment
	start := 0
	if strings.HasPrefix(text, utf8BOM) {
		start = len(utf8BOM)
	}
	for pos := start; pos < len(text); {
		lineEnd := lineEndAt(text, pos)
		line := text[pos:lineEnd]
		next := lineEnd + 1

		if m := assignmentRegex.FindStringSubmatchIndex(line); m != nil {
			a := assignment{
				name:       line[m[4]:m[5]],
				export:     m[2] >= 0,
				lineStart:  pos,
				valueStart: pos + m[1],
			}
			rest := text[a.valueStart:]
			if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, "'") {
				if closing := closingQuote(rest); closing >= 0 {
					a.valueEnd = a.valueStart + closing + 1
					next = lineEndAt(text, a.valueEnd) + 1
				} else {
					a.valueEnd = trimLineEnd(text, a.valueStart, lineEnd)
				}
			} else {
				value := text[a.valueStart:lineEnd]
				if i := strings.Index(value, " #"); i >= 0 {
					value = value[:i]
				}
				a.valueEnd = a.valueStart + len(strings.TrimRight(value, " \t\r"))
			}
			found = append(found, a)
		} else if m := commentedRegex.FindStringSubmatchIndex(line); m != nil {
			found = append(found, assignment{
				name:      line[m[4]:m[5]],
				export:    m[2] >= 0,
				commented: true,
				lineStart: pos,
				valueEnd:  trimLineEnd(text, pos, lineEnd),
			})
		}
		pos = next
	}
	return found
}

// closingQuote returns the index of the quote that closes s[0], skipping
// backslash-escaped characters, or -1 if the value is unterminated.
func closingQuote(s string) int {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case s[0]:
			return i
		}
	}
	return -1
}

func lineEndAt(text string, pos int) int {
	if i := strings.IndexByte(text[pos:], '\n'); i >= 0 {
		return pos + i
	}
	return len(text)
}

// trimLineEnd returns lineEnd moved back past a CR, so CRLF files keep theirs.
func trimLineEnd(text string, start, lineEnd int) int {
	if lineEnd > start && text[lineEnd-1] == '\r' {
		return lineEnd - 1
	}
	return lineEnd
}
