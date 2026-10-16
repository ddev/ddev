// Package dotenv reads, writes and quotes env files.
//
// Values are parsed by the same parser docker compose uses, so a value DDEV
// passes to a container matches what ${VAR} gives in a compose file.
package dotenv

import (
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	composedotenv "github.com/compose-spec/compose-go/v2/dotenv"
)

// Parse parses the text of the env file at path, which only labels errors.
// ${VAR} references resolve through lookup first, then through values defined
// earlier in the text.
func Parse(path, text string, lookup func(string) (string, bool)) (map[string]string, error) {
	values := map[string]string{}
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}
	err := parseInto(values, path, text, lookup)
	return values, err
}

// parseInto parses text into values. compose-go reads a bare NAME on a last
// line with no newline as a value with an empty name, so one is added.
func parseInto(values map[string]string, path, text string, lookup func(string) (string, bool)) error {
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return composedotenv.ParseWithFormat(strings.NewReader(text), path, values, lookup, "")
}

// ReadFiles merges env files in order, later files overriding earlier ones.
// ${VAR} references resolve against the host environment first, then against
// values from earlier files, as compose resolves an env_file list.
func ReadFiles(paths ...string) (map[string]string, error) {
	return readFiles(paths, os.LookupEnv)
}

// ReadNames returns the names that env files set, a bare NAME line included,
// without resolving ${VAR} references, so an unset one neither warns nor fails.
func ReadNames(paths ...string) ([]string, error) {
	values, err := readFiles(paths, func(string) (string, bool) { return "x", true })
	if err != nil {
		return nil, err
	}
	return slices.Sorted(maps.Keys(values)), nil
}

func readFiles(paths []string, lookup func(string) (string, bool)) (map[string]string, error) {
	values := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err = parseInto(values, path, string(data), lookup); err != nil {
			return nil, err
		}
	}
	return values, nil
}

// Quote returns value as an env file value that compose, phpdotenv and
// Symfony Dotenv read back unchanged, with no expansion, as does CakePHP's
// josegonzalez/dotenv for a value with no backslash.
//
// Values are double-quoted, except that a value with `\` or `$` is
// single-quoted when it can be: josegonzalez keeps `\$` as written, and
// Symfony expands \n in `"\\n"` before it unescapes \\.
func Quote(value string) string {
	if strings.ContainsAny(value, `\$`) && !strings.ContainsAny(value, "'\n\r") && !strings.HasSuffix(value, `\`) {
		return "'" + value + "'"
	}
	return `"` + quoteReplacer.Replace(value) + `"`
}

var quoteReplacer = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	`$`, `\$`,
	"\n", `\n`,
	"\r", `\r`,
)

var bashNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// bashReadonlyNames fail `export` under set -e, and UID is a common compose
// variable in .ddev/.env.
var bashReadonlyNames = []string{"BASHOPTS", "BASH_VERSINFO", "EUID", "PPID", "SHELLOPTS", "UID"}

// CanExport reports whether bash can export name. Compose accepts names it
// cannot, such as CodeIgniter's "database.default.hostname".
func CanExport(name string) bool {
	return bashNameRegex.MatchString(name) && !slices.Contains(bashReadonlyNames, name)
}

// ShellExports returns one bash `export NAME='value'` line per variable,
// sorted by name. Values are single-quoted, so bash never expands them.
// Names CanExport rejects are left out.
func ShellExports(values map[string]string) string {
	return shellExports(values, shellQuote)
}

// ShellExportsWithExpansion is ShellExports for values read as the inside of
// a bash double-quoted string, such as provider environment_variables:
// $VAR, ${VAR}, $(cmd) and a leading ~ expand where the script runs, \ escapes
// $ and \, and other characters, quotes and backticks included, are literal.
func ShellExportsWithExpansion(values map[string]string) string {
	return shellExports(values, shellDoubleQuote)
}

// EscapeExpansion escapes $ and \ in value, so ShellExportsWithExpansion
// exports it as written, for a value already taken from the host. A leading ~
// still expands.
func EscapeExpansion(value string) string {
	return expansionEscaper.Replace(value)
}

var expansionEscaper = strings.NewReplacer(`\`, `\\`, `$`, `\$`)

func shellExports(values map[string]string, quote func(string) string) string {
	var lines []string
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !CanExport(name) {
			continue
		}
		lines = append(lines, "export "+name+"="+quote(values[name]))
	}
	return strings.Join(lines, "\n")
}

// shellQuote single-quotes s for bash and sh, which read it back unchanged.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellDoubleQuote double-quotes s, escaping " and backticks unless s
// already escapes them; a $(...) is copied as written, since quotes inside
// it start a new context. A leading ~ or ~/ stays unquoted, because bash only
// expands a tilde with no quote before the first slash.
func shellDoubleQuote(s string) string {
	if s == "~" {
		return s
	}
	tilde := ""
	if strings.HasPrefix(s, "~/") {
		tilde, s = "~/", s[2:]
	}
	var b strings.Builder
	backslashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		escaped := backslashes%2 == 1
		if c == '$' && !escaped && strings.HasPrefix(s[i+1:], "(") {
			end := commandSubstitutionEnd(s, i+1)
			b.WriteString(s[i:end])
			i, backslashes = end-1, 0
			continue
		}
		if (c == '"' || c == '`') && !escaped {
			b.WriteByte('\\')
		}
		if c == '\\' {
			backslashes++
		} else {
			backslashes = 0
		}
		b.WriteByte(c)
	}
	// A trailing lone backslash would escape the closing quote
	if backslashes%2 == 1 {
		b.WriteByte('\\')
	}
	return tilde + `"` + b.String() + `"`
}

// commandSubstitutionEnd returns the index after the ) that closes the ( at
// open, or len(s) when it is unbalanced. A ( or ) inside quotes does not count.
func commandSubstitutionEnd(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '\'':
			if end := strings.IndexByte(s[i+1:], '\''); end >= 0 {
				i += end + 1
			} else {
				i = len(s)
			}
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				switch {
				case s[i] == '\\':
					i++
				case strings.HasPrefix(s[i:], "$("):
					i = commandSubstitutionEnd(s, i+1) - 1
				}
			}
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i + 1
			}
		}
	}
	return len(s)
}
