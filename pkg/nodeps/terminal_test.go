package nodeps_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

// TestTerminalExecEnvForwardsKnownTerm tests that a TERM the container's
// terminfo database can resolve is forwarded unchanged.
func TestTerminalExecEnvForwardsKnownTerm(t *testing.T) {
	require.Equal(t, []string{"TERM=xterm-256color"}, nodeps.TerminalExecEnv(nil, "xterm-256color", ""))
}

// TestTerminalExecEnvNormalizesUnknownTerm tests that a TERM with no terminfo
// entry in the container is replaced by one that resolves there, rather than
// being forwarded and leaving the shell with an unusable TERM.
func TestTerminalExecEnvNormalizesUnknownTerm(t *testing.T) {
	for _, hostTerm := range []string{"xterm-kitty", "alacritty", "wezterm", "xterm-ghostty", "tmux-direct"} {
		require.Equal(t, []string{"TERM=xterm-256color"}, nodeps.TerminalExecEnv(nil, hostTerm, ""), "hostTerm=%s", hostTerm)
	}
}

// TestTerminalExecEnvSkipsUnsetTerm tests that no TERM is forwarded when the
// host has none, so the container keeps its own default.
func TestTerminalExecEnvSkipsUnsetTerm(t *testing.T) {
	require.Empty(t, nodeps.TerminalExecEnv(nil, "", ""))
	require.Equal(t, []string{"XDEBUG_MODE=off"}, nodeps.TerminalExecEnv([]string{"XDEBUG_MODE=off"}, "", "truecolor"))
}

// TestTerminalExecEnvForwardsColorterm tests that COLORTERM is forwarded, as it
// is what tells programs in the container that truecolor is available.
func TestTerminalExecEnvForwardsColorterm(t *testing.T) {
	require.Equal(t, []string{"TERM=xterm-256color", "COLORTERM=truecolor"},
		nodeps.TerminalExecEnv(nil, "xterm-256color", "truecolor"))
}

// TestTerminalExecEnvKeepsExistingEnv tests that variables a caller has set
// explicitly win over the forwarded values of the host.
func TestTerminalExecEnvKeepsExistingEnv(t *testing.T) {
	require.Equal(t, []string{"XDEBUG_MODE=off", "TERM=screen", "COLORTERM=truecolor"},
		nodeps.TerminalExecEnv([]string{"XDEBUG_MODE=off", "TERM=screen"}, "xterm-256color", "truecolor"))
	require.Equal(t, []string{"TERM=screen", "COLORTERM="},
		nodeps.TerminalExecEnv([]string{"TERM=screen", "COLORTERM="}, "xterm-256color", "truecolor"))
}
