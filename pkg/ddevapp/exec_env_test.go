package ddevapp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentDetectionEnv(t *testing.T) {
	unsetAgentDetectionEnv(t)
	t.Setenv("AI_AGENT", "codex")
	t.Setenv("CLAUDE_CODE", "1")
	t.Setenv("COPILOT_GITHUB_TOKEN", "secret")

	env := agentDetectionEnv(nil)
	require.Contains(t, env, "AI_AGENT=codex")
	require.Contains(t, env, "CLAUDE_CODE=1")
	require.NotContains(t, env, "COPILOT_GITHUB_TOKEN=secret")
}

func TestAgentDetectionEnvPreservesExplicitEnv(t *testing.T) {
	unsetAgentDetectionEnv(t)
	t.Setenv("AI_AGENT", "host-agent")

	env := agentDetectionEnv([]string{"AI_AGENT=explicit-agent"})
	require.Empty(t, env)
}

func TestAgentDetectionEnvUnsetVariables(t *testing.T) {
	unsetAgentDetectionEnv(t)

	env := agentDetectionEnv(nil)
	require.Empty(t, env)
}

func unsetAgentDetectionEnv(t *testing.T) {
	t.Helper()

	for _, name := range agentDetectionEnvVars {
		originalValue, wasSet := os.LookupEnv(name)
		require.NoError(t, os.Unsetenv(name))
		if wasSet {
			t.Cleanup(func() {
				require.NoError(t, os.Setenv(name, originalValue))
			})
		}
	}
}
