#!/bin/bash
# SessionStart hook: put the docs tooling and the built ddev binary on PATH for
# every later tool call and hook, the two locations .envrc adds. See
# .claude/README.md.

set -uo pipefail

if [ -z "${CLAUDE_ENV_FILE:-}" ]; then
  printf '{"systemMessage":"session-start: no CLAUDE_ENV_FILE, so the built ddev binary is not on PATH. make targets still find the docs tools on their own."}\n'
  exit 0
fi

build=""
if os=$(go env GOHOSTOS 2>/dev/null) && arch=$(go env GOHOSTARCH 2>/dev/null); then
  build=":${CLAUDE_PROJECT_DIR:-$PWD}/.gotmp/bin/${os}_${arch}"
fi

# Claude Code expands $PATH when it applies the file, so this line is the same
# on every firing and needs no guard against being written twice.
printf 'export PATH="%s%s:$PATH"\n' \
  "$HOME/.ddev-dev-tools/python/bin:$HOME/.ddev-dev-tools/node/bin" "$build" \
  >"$CLAUDE_ENV_FILE"
