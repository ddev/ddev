---
name: ddev-commit
description: Write a DDEV commit message, pull request, or issue. Covers the Conventional Commits format, the required PR template sections, how to pass a body to git and gh without bash mangling it, and the plain-language register DDEV uses for descriptions. Use before writing any commit message, PR, or issue for this repository.
when_to_use: >
  Triggered by "commit this", "commit my changes", "write a commit message",
  "open a PR", "create a pull request", "update the PR description",
  "file an issue", "write up this bug", or any request to prepare work for
  review in the DDEV repo.
---

# Writing commits, PRs, and issues for DDEV

In this repo the initial commit message for a PR *contains* the PR template, so
the commit body and the PR description are the same text. Write it once, well.

## Always pass a body through a file

**Write the message to a file first, then `git commit -F <file>`,** for every
commit. Never use `git commit -m "$(cat <<'EOF' ... EOF)"` — the
inline-heredoc-inside-`$(...)` form is unreliable in bash and mangles the
message; use the Write tool or a plain `cat > file <<'EOF'` command instead.
The same holds for `gh`: `--body-file`, never `--body "$(...)"`.

```bash
gh pr create --title "<type>: <description>" --body-file ~/tmp/pr_body.md
gh issue create --title "..." --body-file ~/tmp/issue_body.md
```

## Commit message format

[Conventional Commits](https://www.conventionalcommits.org/):
`<type>[optional scope][optional !]: <description>[, fixes #<issue>]`. Types:
`build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `style`,
`test` — for example `fix: handle container networking timeout, fixes #1234`.

## The PR template

Mirror `.github/PULL_REQUEST_TEMPLATE.md`. Required sections, in this order:

```markdown
## Short Summary (TL;DR)

## The Issue

- Fixes #REPLACE_ME_WITH_RELATED_ISSUE_NUMBER

## How This PR Solves The Issue

## Manual Testing Instructions

## Automated Testing Overview

## Release/Deployment Notes
```

Per-section notes, in the same top-to-bottom order as the template above:

- **Short Summary (TL;DR):** one or two plain-language sentences — what
  changed and why. Issues follow the same rule for their own `Short summary
  (TL;DR)` field in `.github/ISSUE_TEMPLATE/`.
- **The Issue / How This PR Solves The Issue:** a decision, not a write-up of
  the investigation behind it. Issues get the same treatment in their
  `Describe your solution`/`Describe alternatives` fields, and an alternatives
  list gets one line each — what it is, why it lost.
- **Manual Testing Instructions:** the commands and what to look for, not the
  obvious steps around them.
- **Release/Deployment Notes:** the last section in the template — give it
  real content, or omit it under the same rule as any other section.

After that last section, append two more trailers, each its own line and
neither part of Release/Deployment Notes content: `🤖 Generated with [Claude
Code](https://claude.com/claude-code)`, then a blank line, then
`Co-Authored-By: Claude <model> <noreply@anthropic.com>` naming the model
actually in use for the session (for example `Claude Opus 5`), not a
hardcoded name.

## Keep the whole message short

A reviewer reads the diff anyway — the message earns its place only by saying
what the diff can't: why, and what to check by hand. **A finished message
should read in under a minute.** A section growing past a handful of lines has
started narrating the diff instead of the decision.

- State each fact once, in the section it best belongs to; point back with
  `see above` rather than repeating it
- Omit a section entirely, heading included, when it does not apply — do not
  keep the heading with `None` or invented content underneath

## Register

Plain, direct sentences, each carrying the connective that ties its clauses
together (so, which, before, instead of, because) — not zero, which reads
choppy, and not two-plus stacked with parentheticals, which reads as dense
engineering prose. Read it aloud as if to a teammate. If a parenthetical is
explaining *why* something was done, give that "why" its own clause instead of
dropping it. The forbidden-word list in `AGENTS.md` applies here with full
force.

## No hard line breaks in GitHub bodies

GitHub renders GFM's hard-line-break behavior in issue, PR, and comment
bodies: a lone `\n` inside a paragraph becomes a literal `<br>`. Hand-wrapping
prose — correct in a committed file, which follows CommonMark — produces a
ragged, too-short-lined paragraph when posted through `gh`.

**In a `--body-file`, write each paragraph as one continuous line**, with
blank lines only between paragraphs, headings, and list items. Code blocks,
tables, and committed files are unaffected.

## Before committing

1. Run the tests relevant to the change, for example
   `go test -run TestName ./pkg/[package]`, and fix anything they report
2. Read `git diff --cached` for comments that restate the code, and cut them —
   the rules are in `.claude/rules/comments.md`
3. If amending the last commit rather than writing a new one, re-check every
   claim already in its body against the current diff and code — a claim
   that was true when written can go stale by the time it's amended — and
   correct or remove anything that no longer holds

A Claude Code hook runs `make staticrequired` before every `git commit`; other
agents should run it themselves.

Then commit and stop there — pushing is forbidden, per `AGENTS.md`. Report the
branch state and which commits are unpushed.
