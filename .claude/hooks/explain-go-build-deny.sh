#!/usr/bin/env bash
# PreToolUse hook for Bash commands that run `go build`, which
# .claude/settings.json denies. A bare deny says only that permission was
# refused, so tell Claude why and what to run instead.
#
# Claude Code also runs this hook for any command it can't parse, whatever
# the `if` filter says, so check the command here and let everything else
# through. The deny rule still blocks forms this check misses.
command=$(jq -r '.tool_input.command // empty' 2>/dev/null) || exit 0
# `go build` at the start of the command or of a subcommand, after optional
# VAR=value assignments.
start='(^|[;&|({]|\$\()[[:space:]]*'
assignments='([A-Za-z_][A-Za-z0-9_]*=[^[:space:]]*[[:space:]]+)*'
if ! grep -Eq "${start}${assignments}go[[:space:]]+build([[:space:]]|$)" <<<"$command"; then
  exit 0
fi
cat >&2 <<'EOF'
go build is denied in this repository; the tool is not missing. Build with make, which writes ddev to .gotmp/bin/<os>_<arch>. To check only that code compiles, run go test -run '^$' ./pkg/<package>. See "Building" in AGENTS.md.
EOF
exit 2
