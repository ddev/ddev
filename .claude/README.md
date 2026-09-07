# Claude Code configuration

What is here and why. Guidance for agents lives in `AGENTS.md` at the repo
root — this file describes only the machinery.

```text
.claude/
├── settings.json      # Environment, permissions, hooks
├── rules/             # Guidance that loads only for matching files
│   ├── go.md              # paths: **/*.go
│   ├── docs.md            # paths: docs/**, *.md
│   ├── comments.md        # paths: **/*
│   └── webserver-config.md
├── skills/
│   └── ddev-commit/       # /ddev-commit: commits, PRs, issues
└── hooks/
    ├── session-start.sh
    └── format-edited-file.sh
```

## How guidance is split

Claude Code reads `CLAUDE.md`, not `AGENTS.md`, so `CLAUDE.md` is a stub that
imports it and adds the few Claude-only notes. The rest follows from how much
context each mechanism costs:

| Mechanism | Loaded | Holds |
| --- | --- | --- |
| `settings.json` | Enforced by the client | Rules that must not depend on Claude choosing to follow them |
| `AGENTS.md` | Every session | Facts that apply to every change |
| `rules/*.md` | When a matching file is read | Guidance for one area of the tree |
| `skills/*/SKILL.md` | When invoked or judged relevant | Procedures you run occasionally |

Adding to `AGENTS.md` costs context in every session, so prefer a rule or a
skill unless the guidance applies to everything.

## Hooks

**`session-start.sh`** writes one `export PATH="..."` line to
`$CLAUDE_ENV_FILE`, putting the docs tooling from
`scripts/install-dev-tools.sh` and the built `ddev` binary into every later
Bash call and hook, so Claude Code need not be launched from a direnv-active
shell.

It names those two locations itself rather than asking direnv, which would
require every contributor to have direnv installed and to have run
`direnv allow` here — and a blocked `.envrc` is invisible, since `direnv exec`
on one exits 0 and applies nothing.

Constraints on this hook, all found the hard way:

- `$CLAUDE_ENV_FILE` is **parsed, not sourced**, and read **once**, right after
  the hook exits. One `export KEY="VALUE"` per line applies, so a multi-line or
  semicolon-joined value is read as one malformed entry and silently dropped.
  Rewriting the file later in the session changes nothing.
- `$PATH` inside a value **is** expanded when the file is applied, so the hook
  writes one constant line and never reads the session's own PATH.
- Values in a `settings.json` `env` block, by contrast, are **literal** —
  `$HOME` and `$PATH` arrive unexpanded — so PATH cannot be set there.
- It carries **no matcher**: a `startup` matcher misses the `resume` that an
  editor restart performs, leaving the line out of the session actually
  running.
- `$CLAUDE_ENV_FILE` is undocumented, so treat it as best-effort. Failures
  report through a JSON `systemMessage`, because a hook that exits 0 has its
  stderr swallowed outside debug mode and plain stdout would land in Claude's
  context instead. `make staticrequired` keeps working regardless, because the
  `Makefile` prepends the dev-tools directories itself.

**Formatting** runs `format-edited-file.sh` on every `Edit`/`Write`. The edited
path comes from the stdin payload, the only place a hook is given it:
`$CLAUDE_FILE_PATHS`, which the hooks guide once documented, is never populated
([anthropics/claude-code#9567](https://github.com/anthropics/claude-code/issues/9567)),
so the hooks written against it formatted nothing for eight months. The path
reaches `make golangci-lint-fix DDEV_GO_FILES=...` or
`make markdownlint-fix DDEV_MD_FILES=...`, so a single edit lints that file,
not the tree. Going through the Makefile keeps every formatter invocation in
one place and picks up the dev-tools PATH it prepends, so formatting does not
depend on the SessionStart export having applied. Neither fix target swallows
its exit status, and an empty path is reported rather than skipped, so a crash
or a broken config is visible.

The script drops any path outside `$CLAUDE_PROJECT_DIR` before it reaches
`DDEV_MD_FILES`: markdownlint-cli crashes outright on a path it cannot express
relative to its working directory (`make -C` puts that at the project root),
which a stray edit outside the repo — the scratchpad, say — would otherwise
trigger. `DDEV_GO_FILES`/`DDEV_MD_FILES` left unset, as `make staticrequired`
and a bare `make golangci-lint-fix` do, fall back to the whole-tree scope the
check targets use.

**The commit gate** is the one hook still an inline `make` call in
`settings.json`, matched by an `if` field rather than a script. It runs
`make staticrequired` on `Bash(git commit *)`; only exit 2 stops a tool call,
and any other non-zero code prints to stderr and lets the commit through.

Neither script looks for a tool anywhere but PATH, and neither needs to, since
`make` prepends the dev-tools directories itself. When the PATH export does
fail, `session-start.sh` says so, and a second route to the same tools would
have hidden that.

## Permissions

The allow list holds only what Claude Code does not already treat as
read-only. `ls`, `cat`, `grep`, `find`, and the read-only `git` forms need no
rule, apart from a few forms that always prompt, such as `find -delete`.

Every rule has a space before its `*`, which matters: `Bash(tr *)` matches only
`tr`, while `Bash(tr*)` would also match `truncate`.

`curl` is not allowed against specific hosts. Argument-matching rules like
`Bash(curl*github.com*)` do not survive a reordered flag, an `https` scheme, a
redirect, or a URL in a variable, so fetching goes through named
`WebFetch(domain:...)` entries instead.

Personal overrides belong in `.claude/settings.local.json`, which is
gitignored. Everything else here is shared team configuration and is committed.
