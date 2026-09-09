package util_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestTerminalExecEnv tests that a TERM the container can resolve is forwarded
// unchanged, that one it cannot is replaced, that an unset TERM leaves the
// container with its own default, that the variables a terminal identifies
// itself with come along, and that the env of the caller wins.
func TestTerminalExecEnv(t *testing.T) {
	testCases := []struct {
		hostTerm        string
		hostColorterm   string
		hostTermProgram string
		existingEnv     []string
		expected        []string
	}{
		{"xterm-256color", "", "", nil, []string{"TERM=xterm-256color"}},
		{"xterm-256color", "truecolor", "", nil, []string{"TERM=xterm-256color", "COLORTERM=truecolor"}},
		{"xterm-256color", "truecolor", "ghostty", nil, []string{"TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=ghostty"}},
		{"screen", "", "", nil, []string{"TERM=screen"}},
		{"xterm-kitty", "", "", nil, []string{"TERM=xterm-256color"}},
		{"alacritty", "", "", nil, []string{"TERM=xterm-256color"}},
		{"wezterm", "", "", nil, []string{"TERM=xterm-256color"}},
		// tmux is in the database but tmux-direct is not, so a prefix match
		// would be wrong here.
		{"tmux-direct", "", "", nil, []string{"TERM=xterm-256color"}},
		{"xterm-ghostty", "truecolor", "ghostty", nil, []string{"TERM=xterm-256color", "COLORTERM=truecolor", "TERM_PROGRAM=ghostty"}},
		// Without a TERM on the host there is nothing to forward.
		{"", "truecolor", "ghostty", nil, nil},
		{"", "truecolor", "", []string{"XDEBUG_MODE=off"}, []string{"XDEBUG_MODE=off"}},
		// The env of the caller survives the merge, as ddev composer relies on
		// for XDEBUG_MODE, and wins over the host.
		{"xterm-256color", "truecolor", "", []string{"XDEBUG_MODE=off"}, []string{"TERM=xterm-256color", "COLORTERM=truecolor", "XDEBUG_MODE=off"}},
		{"xterm-256color", "truecolor", "", []string{"TERM=vt100"}, []string{"TERM=vt100", "COLORTERM=truecolor"}},
		{"xterm-256color", "truecolor", "", []string{"COLORTERM="}, []string{"TERM=xterm-256color", "COLORTERM="}},
	}

	// The terminal running the tests must not leak into the cases.
	for _, varName := range util.TerminalEnvVars {
		t.Setenv(varName, "")
	}
	for _, tc := range testCases {
		t.Setenv("TERM", tc.hostTerm)
		t.Setenv("COLORTERM", tc.hostColorterm)
		t.Setenv("TERM_PROGRAM", tc.hostTermProgram)
		// EnvToUniqueEnv does not keep the order, so compare the elements.
		require.ElementsMatch(t, tc.expected, util.TerminalExecEnv(tc.existingEnv), "hostTerm=%s existingEnv=%v", tc.hostTerm, tc.existingEnv)
	}
}
