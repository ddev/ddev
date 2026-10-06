package ddevapp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetComposerEnv tests that XDEBUG_MODE=off always comes first, and that
// the Composer variables set on the host follow it, empty ones included.
func TestGetComposerEnv(t *testing.T) {
	for _, name := range composerEnvVars {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	require.Equal(t, []string{"XDEBUG_MODE=off"}, getComposerEnv())

	t.Setenv("COMPOSER_NO_BLOCKING", "1")
	t.Setenv("COMPOSER_NO_SECURITY_BLOCKING", "")
	require.Equal(t, []string{"XDEBUG_MODE=off", "COMPOSER_NO_BLOCKING=1", "COMPOSER_NO_SECURITY_BLOCKING="}, getComposerEnv())
}
