package ddevapp

import (
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/util"
)

// aiAgentEnvVars are the variables AI coding agents set to identify
// themselves, which tools like PHPStan read to adjust their output. Forwarding
// them lets a tool in the container see the agent that ran `ddev exec`.
// COPILOT_GITHUB_TOKEN is left out because it is a credential.
// See https://github.com/laravel/agent-detector#supported-agents, which this
// list matches as of commit 943c080.
var aiAgentEnvVars = []string{
	"AI_AGENT",
	"AMP_CURRENT_THREAD_ID",
	"ANTIGRAVITY_AGENT",
	"ANTIGRAVITY_CLI_ALIAS",
	"AUGMENT_AGENT",
	"CLAUDECODE",
	"CLAUDE_CODE",
	"CLAUDE_CODE_IS_COWORK",
	"CLINE_ACTIVE",
	"CODEX_CI",
	"CODEX_SANDBOX",
	"CODEX_SANDBOX_NETWORK_DISABLED",
	"CODEX_THREAD_ID",
	"COPILOT_ALLOW_ALL",
	"COPILOT_CLI",
	"COPILOT_MODEL",
	"CURSOR_AGENT",
	"GEMINI_CLI",
	"GOOSE_TERMINAL",
	"GROK_PLUGIN_DATA",
	"GROK_PLUGIN_ROOT",
	"JUNIE_DATA",
	"JUNIE_SHIM_PATH",
	"KIMI_PLUGIN_ROOT",
	"KIRO_AGENT_PATH",
	"MATTERHORN_SESSION_ID",
	"OPENCLAW_SHELL",
	"OPENCODE",
	"OPENCODE_CLIENT",
	"PI_CODING_AGENT",
	"REPL_ID",
}

// execEnv returns the environment of a command run in service by app.Exec,
// built from host variables and callerEnv. A later source overrides an
// earlier one: project config first, then what describes the calling
// session, so a sanitized TERM wins over a bare TERM in web_environment,
// and callerEnv wins over everything.
func (app *DdevApp) execEnv(service string, tty bool, callerEnv []string) []string {
	var sources [][]string
	if service == "web" {
		sources = append(sources, app.webEnvironmentHostEnv())
	}
	sources = append(sources, util.HostEnv(aiAgentEnvVars...))
	// Without a TTY there is no terminal to describe, and programs write
	// plain text anyway.
	if tty {
		sources = append(sources, util.TerminalEnv())
	}
	return util.MergeEnv(append(sources, callerEnv)...)
}

// webEnvironmentHostEnv returns the current host value of each bare name in
// web_environment. Compose resolves those only when the container starts, so
// a variable set after `ddev start`, as an AI agent does, would never arrive.
// A name an env file sets for web is skipped, because that file overrides
// web_environment in the container, and exec has to agree with it.
func (app *DdevApp) webEnvironmentHostEnv() []string {
	var names []string
	for _, entry := range app.webEnvironment() {
		if !strings.Contains(entry, "=") {
			names = append(names, entry)
		}
	}
	// Internal execs run often, so read the env files only when needed.
	if len(names) == 0 {
		return nil
	}
	envFileVars, err := app.ReadEnvFilesForTarget("web")
	if err != nil {
		util.Debug("Not checking env files for web_environment names on exec: %v", err)
	}
	names = slices.DeleteFunc(names, func(name string) bool {
		_, inEnvFile := envFileVars[name]
		return inEnvFile
	})
	return util.HostEnv(names...)
}
