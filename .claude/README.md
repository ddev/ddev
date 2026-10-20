# Claude Code configuration notes

Maintainer notes on the machinery; guidance for agents lives in `AGENTS.md`.

## Where guidance goes

| Mechanism | Loaded | Holds |
| --- | --- | --- |
| `settings.json` | Enforced by the client | Rules that must not depend on Claude choosing to follow them |
| `AGENTS.md` | Every session | Facts that apply to every change |
| `rules/*.md` | When a file matching `paths:` is read, or always without it | Guidance for one area of the tree |
| `skills/*/SKILL.md` | When invoked or judged relevant | Procedures used occasionally |

`AGENTS.md` costs context in every session, so prefer a rule or a skill unless
the guidance applies to everything.

## Hooks

Hooks run in Claude's current directory, so each `make` call passes
`-C "$CLAUDE_PROJECT_DIR"`.

The formatters read the edited path from the stdin payload with `jq`, the only
place a hook is given it: `$CLAUDE_FILE_PATHS` is never populated
([anthropics/claude-code#9567](https://github.com/anthropics/claude-code/issues/9567)).
Their output goes to stderr because on a lint error make exits 2, and Claude
Code shows only stderr to Claude.

A hook's `if` takes exactly one permission rule. `Edit(...)` does not match
the Write tool, and `||` or `{a,b}` inside a rule matched nothing in testing,
so each tool and file type gets its own entry. The leading `/` anchors a
pattern to the project, so edits outside it are not formatted.

There is no commit hook. Matching `git commit` in Bash text either misses
forms like `git -C dir commit` or catches read-only commands, so the commit
skill runs `make staticrequired` instead.

SessionStart writes PATH to `$CLAUDE_ENV_FILE`, which Claude Code sources
before each Bash command. PATH cannot go in the `env` block, whose values are
literal: `$PATH` and `$HOME` arrive unexpanded. `make` does not depend on this
hook, since the `Makefile` prepends the dev-tools directories itself.

## Permissions

Every rule has a space before its `*`: `Bash(tr *)` matches only `tr`, while
`Bash(tr*)` would also match `truncate`. Fetching is allowed per
`WebFetch(domain:...)` rather than by `curl` argument patterns, which a
reordered flag or a URL in a variable defeats.

## Instruction loading

A project or parent `CLAUDE.local.md` can stop Claude from loading `AGENTS.md`
under the default instruction setting. Use an `@AGENTS.md` import in a
`CLAUDE.md` or select the setting that reads both instruction files.

References: [hooks](https://code.claude.com/docs/en/hooks),
[file rules](https://code.claude.com/docs/en/permissions#read-and-edit),
[instruction loading](https://code.claude.com/docs/en/memory#when-claude-code-reads-agentsmd).
