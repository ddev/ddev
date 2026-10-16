// Package dotenv reads, writes and quotes env files.
//
// Values are parsed by the same parser docker compose uses, so a value DDEV
// passes to a container matches what ${VAR} gives in a compose file.
package dotenv

import (
	"bytes"
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
	err := composedotenv.ParseWithFormat(strings.NewReader(text), path, values, lookup, "")
	return values, err
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
		if err = composedotenv.ParseWithFormat(bytes.NewReader(data), path, values, lookup, ""); err != nil {
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
