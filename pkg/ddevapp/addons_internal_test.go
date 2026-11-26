package ddevapp

import (
	"path/filepath"
	"testing"

	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestBashExportLine checks that every .env value reaches a bash action unchanged.
func TestBashExportLine(t *testing.T) {
	require.Empty(t, bashExportLine(nil))
	require.Empty(t, bashExportLine(map[string]string{"FOO.BAR": "secret", "1X": "secret"}))
	require.Equal(t, "export OK='1'", bashExportLine(map[string]string{"FOO-BAR": "secret", "OK": "1"}))

	values := map[string]string{
		"PARENS":    "ab(c)d",
		"SEMICOLON": "a;touch injected",
		"PIPE_AMP":  "a|b&c<d>e",
		"QUOTES":    `it's "quoted"`,
		"DOLLAR":    "$HOME ${PATH} $(id) `id`",
		"SPACES":    "  leading and trailing  ",
		"BACKSLASH": `a\b\\c\`,
		"NEWLINE":   "line1\nline2",
		"TEMPLATE":  "{{ .DdevProjectConfig.name }}",
		"GLOB":      "*?[a]~",
		"EMPTY":     "",
	}
	line := bashExportLine(values)
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	for k, want := range values {
		out, err := exec.RunHostCommand(util.FindBashPath(), "-c", "set -eu -o pipefail\n"+line+"\nprintf '%s' \"$"+k+"\"")
		require.NoError(t, err, "output=%s", out)
		require.Equal(t, want, out, "value of %s", k)
	}
	require.NoFileExists(t, filepath.Join(tmpDir, "injected"))
}
