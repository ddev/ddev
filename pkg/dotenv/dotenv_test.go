package dotenv_test

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/dotenv"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// trickyValues covers characters that have broken env handling before.
var trickyValues = map[string]string{
	"PAREN":            ")value(",
	"SHELL_OPS":        "a;b&c|d<e>f",
	"BACKTICK":         "a`id`b",
	"SUBSHELL":         "$(id)",
	"DOLLAR":           "$HOME ${HOME} $$",
	"DEFAULT_SYNTAX":   "${UNSET:-def}",
	"SQUOTE":           "it's",
	"DQUOTE":           `say "hi"`,
	"TRAILING_DQUOTE":  `ends with "`,
	"ONLY_DQUOTES":     `""`,
	"BACKSLASH":        `C:\new\table\back`,
	"TRAILING_BSLASH":  `ends with \`,
	"ESCAPED_DOLLAR":   `\$x`,
	"LITERAL_ESCAPES":  `\n\t\r\\`,
	"NEWLINE":          "line1\nline2",
	"CRLF":             "line1\r\nline2",
	"TAB":              "a\tb",
	"HASH":             "#not a comment",
	"SPACE_HASH":       "value #not a comment",
	"TILDE":            "~/dir",
	"BRACES":           "a{1,2}b",
	"TEMPLATE":         "{{ .Name }}",
	"GLOB":             "*",
	"SPACES":           "  two  spaces  ",
	"EQUALS":           "a=b=c",
	"UNICODE":          "héllo wörld ✓",
	"EMPTY":            "",
	"export_lowercase": "x",
}

func TestQuoteRoundTrip(t *testing.T) {
	var text strings.Builder
	for _, name := range slices.Sorted(maps.Keys(trickyValues)) {
		text.WriteString(name + "=" + dotenv.Quote(trickyValues[name]) + "\n")
	}
	got, err := dotenv.Parse(".env", text.String(), nil)
	require.NoError(t, err)
	require.Equal(t, trickyValues, got)
}

func TestQuoteFormat(t *testing.T) {
	require.Equal(t, `"plain"`, dotenv.Quote("plain"))
	require.Equal(t, `"say \"hi\""`, dotenv.Quote(`say "hi"`))
	require.Equal(t, `'C:\new $x "y"'`, dotenv.Quote(`C:\new $x "y"`))
	require.Equal(t, `'$dollar_variable'`, dotenv.Quote(`$dollar_variable`))
	require.Equal(t, `"it's C:\\new \$x"`, dotenv.Quote(`it's C:\new $x`))
	require.Equal(t, `"a\\b \"c\" \$d\ne"`, dotenv.Quote("a\\b \"c\" $d\ne"))
	require.Equal(t, `"ends with \\"`, dotenv.Quote(`ends with \`))
}

func TestShellExportsWithExpansion(t *testing.T) {
	t.Setenv("DOTENV_TEST_VAR", "from-env")
	home, err := exec.Command(util.FindBashPath(), "-c", `printf '%s' "$HOME"`).Output()
	require.NoError(t, err)
	rootHome, err := exec.Command(util.FindBashPath(), "-c", `printf '%s' ~root`).Output()
	require.NoError(t, err)

	tests := map[string]struct{ value, want string }{
		"PLAIN":          {"plain", "plain"},
		"SPACES":         {"two  words", "two  words"},
		"SHELL_OPS":      {")a;b&c|d<e>f(", ")a;b&c|d<e>f("},
		"SQUOTE":         {"it's", "it's"},
		"DQUOTE":         {`say "hi"`, `say "hi"`},
		"ESCAPED_DQUOTE": {`say \"hi\"`, `say "hi"`},
		"BACKTICK":       {"a`id`b", "a`id`b"},
		"WINDOWS_PATH":   {`C:\path\to`, `C:\path\to`},
		"TRAILING_BS":    {`ends with \`, `ends with \`},
		"NEWLINE":        {"line1\nline2", "line1\nline2"},
		"BS_NEWLINE":     {"line1\\\nline2", "line1\\\nline2"},
		"VAR":            {"$DOTENV_TEST_VAR/x", "from-env/x"},
		"BRACED_VAR":     {"${DOTENV_TEST_VAR}-x", "from-env-x"},
		"DEFAULT":        {"${DOTENV_TEST_UNSET:-def}", "def"},
		"DEFAULT_QUOTED": {`${DOTENV_TEST_UNSET:-"a b"} "c"`, `a b "c"`},
		"DEFAULT_NESTED": {`${DOTENV_TEST_UNSET:-${DOTENV_TEST_VAR}}}`, "from-env}"},
		"ESCAPED_DOLLAR": {`\$DOTENV_TEST_VAR`, "$DOTENV_TEST_VAR"},
		"SUBSHELL":       {"$(echo sub)", "sub"},
		"SUBSHELL_QUOTE": {`$(echo "x y") "z"`, `x y "z"`},
		"SUBSHELL_NEST":  {`a$(echo "$(echo in)")b`, "ainb"},
		"SUBSHELL_PAREN": {`$(echo ")" ')(')x`, ") )(x"},
		"ARITHMETIC":     {"$((2+3))", "5"},
		"UNC_PATH":       {`\\server\share`, `\server\share`},
		"TILDE":          {"~", string(home)},
		"TILDE_PATH":     {"~/tmp/repo", string(home) + "/tmp/repo"},
		"TILDE_INSIDE":   {"a~b ~/x", "a~b ~/x"},
		"TILDE_USER":     {"~root/x", string(rootHome) + "/x"},
		"EMPTY":          {"", ""},
	}
	if nodeps.IsWindows() {
		// Git Bash halves \\ in a bash -c argument, so this value loses its
		// backslash and the line continues. Provider host commands share this.
		delete(tests, "BS_NEWLINE")
	}
	values := map[string]string{}
	for name, tc := range tests {
		values[name] = tc.value
	}
	names := slices.Sorted(maps.Keys(values))
	script := dotenv.ShellExportsWithExpansion(values) + "\nprintf '%s\\0'"
	for _, name := range names {
		script += ` "$` + name + `"`
	}
	out, err := exec.Command(util.FindBashPath(), "-c", script).Output()
	require.NoError(t, err, "script=%s", script)
	got := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	require.Len(t, got, len(names))
	for i, name := range names {
		require.Equal(t, tests[name].want, got[i], "value of %s", name)
	}
}

func TestEscapeExpansion(t *testing.T) {
	values := []string{`pa$$w`, "a`id`\"b\"", `ends with \`, `$HOME/~`}
	if !nodeps.IsWindows() {
		// Git Bash halves the escaped \\ in a bash -c argument, so $(id) runs
		values = append(values, `a\$(id)b`)
	}
	for _, value := range values {
		script := dotenv.ShellExportsWithExpansion(map[string]string{"V": dotenv.EscapeExpansion(value)}) + "\nprintf '%s' \"$V\""
		out, err := exec.Command(util.FindBashPath(), "-c", script).Output()
		require.NoError(t, err, "script=%s", script)
		require.Equal(t, value, string(out), "script=%s", script)
	}
}

func TestShellExportsSorted(t *testing.T) {
	values := map[string]string{"B": "2", "A": "1", "my-var": "x", "UID": "1000", "database.default.hostname": "db"}
	require.Equal(t, "export A=\"1\"\nexport B=\"2\"", dotenv.ShellExportsWithExpansion(values))
	require.Empty(t, dotenv.ShellExportsWithExpansion(nil))
}

func TestReadFiles(t *testing.T) {
	t.Setenv("DOTENV_TEST_HOST", "from-host")
	t.Setenv("DOTENV_TEST_SHADOWED", "from-host")
	t.Setenv("DOTENV_TEST_BARE", "bare-from-host")
	dir := t.TempDir()
	first := filepath.Join(dir, ".env")
	second := filepath.Join(dir, ".env.web")
	require.NoError(t, os.WriteFile(first, []byte(`SHARED=first
FIRST_ONLY=one
my-var=hyphen
database.default.hostname=db
DOTENV_TEST_SHADOWED=from-file
`), 0644))
	require.NoError(t, os.WriteFile(second, []byte("SHARED=second\r\n"+
		`FROM_FIRST="${FIRST_ONLY}-two"`+"\r\n"+
		`FROM_HOST="${DOTENV_TEST_HOST}"`+"\r\n"+
		`HOST_WINS="${DOTENV_TEST_SHADOWED}"`+"\r\n"+
		`DEFAULTED="${DOTENV_TEST_UNSET:-fallback}"`+"\r\n"+
		`SINGLE='${FIRST_ONLY}'`+"\r\n"+
		`ESCAPED="\${FIRST_ONLY}"`+"\r\n"+
		`export EXPORTED=yes`+"\r\n"+
		"DOTENV_TEST_BARE\r\n"+
		"DOTENV_TEST_BARE_UNSET\r\n"), 0644))

	got, err := dotenv.ReadFiles(first, second)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"SHARED":                    "second",
		"FIRST_ONLY":                "one",
		"my-var":                    "hyphen",
		"database.default.hostname": "db",
		"DOTENV_TEST_SHADOWED":      "from-file",
		"FROM_FIRST":                "one-two",
		"FROM_HOST":                 "from-host",
		"HOST_WINS":                 "from-host",
		"DEFAULTED":                 "fallback",
		"SINGLE":                    "${FIRST_ONLY}",
		"ESCAPED":                   "${FIRST_ONLY}",
		"EXPORTED":                  "yes",
		"DOTENV_TEST_BARE":          "bare-from-host",
	}, got)

	names, err := dotenv.ReadNames(first, second)
	require.NoError(t, err)
	require.Equal(t, slices.Sorted(slices.Values(append(slices.Collect(maps.Keys(got)), "DOTENV_TEST_BARE_UNSET"))), names)

	noNewline := filepath.Join(dir, ".env.no-newline")
	require.NoError(t, os.WriteFile(noNewline, []byte("A=1\nDOTENV_TEST_BARE"), 0644))
	got, err = dotenv.ReadFiles(noNewline)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"A": "1", "DOTENV_TEST_BARE": "bare-from-host"}, got)

	_, err = dotenv.ReadFiles(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)

	broken := filepath.Join(dir, ".env.broken")
	require.NoError(t, os.WriteFile(broken, []byte(`BROKEN="unterminated`+"\n"), 0644))
	_, err = dotenv.ReadFiles(broken)
	require.ErrorContains(t, err, broken)
}
