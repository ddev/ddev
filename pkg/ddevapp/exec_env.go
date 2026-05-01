package ddevapp

import (
	"os"
	"strings"
)

var agentDetectionEnvVars = []string{
	"AI_AGENT",
	"CURSOR_AGENT",
	"GEMINI_CLI",
	"CODEX_SANDBOX",
	"CODEX_CI",
	"CODEX_THREAD_ID",
	"AUGMENT_AGENT",
	"AMP_CURRENT_THREAD_ID",
	"OPENCODE_CLIENT",
	"OPENCODE",
	"CLAUDECODE",
	"CLAUDE_CODE",
	"CLAUDE_CODE_IS_COWORK",
	"COPILOT_MODEL",
	"COPILOT_ALLOW_ALL",
	"COPILOT_CLI",
	"REPL_ID",
	"ANTIGRAVITY_AGENT",
	"PI_CODING_AGENT",
	"KIRO_AGENT_PATH",
}

func agentDetectionEnv(existingEnv []string) []string {
	seen := make(map[string]bool, len(existingEnv))
	for _, envVar := range existingEnv {
		name, _, _ := strings.Cut(envVar, "=")
		seen[name] = true
	}

	env := make([]string, 0, len(agentDetectionEnvVars))
	for _, name := range agentDetectionEnvVars {
		if seen[name] {
			continue
		}
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}
