package util_test

import (
	"os"
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestTerminalEnv tests that a TERM the container can resolve is forwarded
// unchanged, that one it cannot is replaced, that an unset TERM leaves the
// container with its own default, and that the variables a terminal
// identifies itself with come along.
func TestTerminalEnv(t *testing.T) {
	testCases := []struct {
		hostTerm        string
		hostColorterm   string
		hostTermProgram string
		expected        []string
	}{
		{"xterm-256color", "", "", []string{"TERM=xterm-256color"}},
		{"xterm-256color", "truecolor", "", []string{"TERM=xterm-256color", "COLORTERM=truecolor"}},
		{"xterm-256color", "truecolor", "ghostty", []string{"TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=ghostty"}},
		{"screen", "", "", []string{"TERM=screen"}},
		{"xterm-kitty", "", "", []string{"TERM=xterm-256color"}},
		{"alacritty", "", "", []string{"TERM=xterm-256color"}},
		{"wezterm", "", "", []string{"TERM=xterm-256color"}},
		// tmux is in the database but tmux-direct is not, so a prefix match
		// would be wrong here.
		{"tmux-direct", "", "", []string{"TERM=xterm-256color"}},
		{"xterm-ghostty", "truecolor", "ghostty", []string{"TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=ghostty"}},
		// Without a TERM on the host there is nothing to forward.
		{"", "truecolor", "ghostty", nil},
	}

	// The terminal running the tests must not leak into the cases. t.Setenv
	// registers the restore that the os.Unsetenv calls below rely on.
	for _, varName := range append([]string{"TERM"}, util.TerminalEnvVars...) {
		t.Setenv(varName, "")
		require.NoError(t, os.Unsetenv(varName))
	}
	for _, tc := range testCases {
		for name, value := range map[string]string{"TERM": tc.hostTerm, "COLORTERM": tc.hostColorterm, "TERM_PROGRAM": tc.hostTermProgram} {
			if value == "" {
				require.NoError(t, os.Unsetenv(name))
			} else {
				t.Setenv(name, value)
			}
		}
		require.Equal(t, tc.expected, util.TerminalEnv(), "hostTerm=%s", tc.hostTerm)
	}
}
