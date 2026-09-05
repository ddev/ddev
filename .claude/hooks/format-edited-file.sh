#!/bin/bash
# PostToolUse(Edit|Write) formatter for the edited file only. Its path comes
# from the stdin payload, the only place a hook is given it.
# Constraints behind the rest: .claude/README.md.

set -uo pipefail

payload=$(cat)

if command -v jq >/dev/null 2>&1; then
  file=$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty')
else
  file=$(printf '%s' "$payload" | grep -o '"file_path":"[^"]*"' | head -1 | cut -d'"' -f4)
fi

# Fail rather than skip: an empty path here is what made an earlier version of
# this hook a silent no-op.
if [ -z "$file" ]; then
  echo "format-edited-file: no file path in the payload, so nothing was formatted" >&2
  exit 1
fi

# Failure exits 1, not make's own 2, which PostToolUse reads as blocking.
case "$file" in
  "$CLAUDE_PROJECT_DIR"/*.go)
    dir="./${file#"$CLAUDE_PROJECT_DIR"/}"
    make -C "$CLAUDE_PROJECT_DIR" golangci-lint-fix DDEV_GO_FILES="${dir%/*}" || exit 1
    ;;
  "$CLAUDE_PROJECT_DIR"/*.md)
    make -C "$CLAUDE_PROJECT_DIR" markdownlint-fix DDEV_MD_FILES="$file" || exit 1
    ;;
esac
