---
name: ddev-commit
description: Write or revise a DDEV commit message, pull request, or issue. Use before creating or editing that text in this repository.
---

# Writing commits, PRs, and issues for DDEV

For an initial commit intended to become a PR, use the PR template in its
body so the same text can serve as the PR description. Issues use the relevant
template under `.github/ISSUE_TEMPLATE/`.

## Always pass a body through a file

**Write the message to a file first, then `git commit -F <file>`,** for every
commit. A file avoids shell interpolation and keeps multiline text reviewable;
use the Write tool or a plain `cat > file <<'EOF'` command.
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

Read `.github/PULL_REQUEST_TEMPLATE.md` before writing. Its sections are in
this order; include the required summary and any other applicable sections:

```markdown
## Short Summary (TL;DR)

## The Issue

- Fixes #REPLACE_ME_WITH_RELATED_ISSUE_NUMBER

## How This PR Solves The Issue

## Manual Testing Instructions

## Automated Testing Overview

## Release/Deployment Notes
```

Per-section notes:

- **Short Summary (TL;DR):** one or two plain-language sentences — what
  changed and why. Issues follow the same rule for their own `Short summary
  (TL;DR)` field in `.github/ISSUE_TEMPLATE/`.
- **The Issue / How This PR Solves The Issue:** describe the problem and the
  chosen solution. Link a related issue when one exists; remove the placeholder
  when none does. Issues get the same treatment in their
  `Describe your solution`/`Describe alternatives` fields, and an alternatives
  list gets one line each — what it is, why it lost.
- **Manual Testing Instructions:** the commands and what to look for, not the
  obvious steps around them. Point the reviewer at
  `ddev utility download-ddev --pr <number>` to get the binary this PR built;
  it needs no Go toolchain, unlike cloning and running `make`. Use the existing
  PR number, or look up the next number for a new PR:
  `gh api 'repos/ddev/ddev/issues?state=all&per_page=1' --jq '.[0].number + 1'`.
  Write the steps as an explicit, reproducible
  transcript with real values (an actual project type, actual commands, the
  actual output or behavior that confirms the fix), not a description of the
  general area to poke at. For example:

  ```bash
  ddev utility download-ddev --pr 8820
  export PATH="$HOME/tmp/ddev-download-ddev/pr-8820:$PATH"
  ddev config --project-type=drupal11 --docroot=web
  ddev start
  ddev snapshot --list
  # Expect the size and DB version columns added by this PR
  ```

  For changes to published docs, link the changed page in the Read the Docs preview
  of the PR, `https://ddev--<number>.org.readthedocs.build/en/<number>/`
  followed by the page path and anchor, for example
  `https://ddev--8820.org.readthedocs.build/en/8820/users/usage/commands/#snapshot`.
  When the steps to test are in that page, tell the reviewer to follow them
  there instead of repeating them in the PR.

- **Release/Deployment Notes:** give it real content, or omit it.

For Claude Code contributions, append `🤖 Developed with assistance from
[Claude Code](https://claude.ai/code)` as its own paragraph after the template,
then the commit trailer `Co-Authored-By: Claude <model>
<noreply@anthropic.com>`. Keep each on one line in the output and use the model
identifier provided by the session. Other agents must use their actual
identity and documented attribution, without inventing a model or email.

## Keep the whole message short

A reviewer reads the diff anyway — the message earns its place only by saying
what the diff can't: why, and what to check by hand. **A finished message
should read in under a minute.** A section growing past a handful of lines has
started narrating the diff instead of the decision.

- State each fact once, in the section it best belongs to; point back with
  `see above` rather than repeating it
- Keep required template fields. Omit optional sections when they do not
  apply; do not invent content to fill them

## Register

Many readers are not native English speakers. Check these rules on reread;
the first four come from
[ASD-STE100](https://en.wikipedia.org/wiki/Simplified_Technical_English):

- At most 20 words in an instruction, and 25 in a description
- Active voice that names who acts: "Claude runs the check", not "the check
  is run"
- One word for one meaning: keep one name for a thing throughout
- One action per step in Manual Testing Instructions
- One connective per sentence (so, which, because): none reads choppy, and
  more stack into dense prose
- Give a "why" its own clause, not a parenthetical

The forbidden-word list in `AGENTS.md` applies here with full force.

## No hard line breaks in GitHub bodies

GitHub turns a lone newline in an issue, PR, or comment body into `<br>`.
**In a `--body-file`, write each paragraph as one continuous line**, with
blank lines only between paragraphs, headings, and list items. Code blocks,
tables, and committed files are unaffected.

## Before committing

1. Run the tests relevant to the change, for example
   `go test -run TestName ./pkg/[package]`, and fix anything they report
2. Read `git diff --cached` for comments that restate the code, and cut them —
   the rules are in `AGENTS.md`
3. **Run `make staticrequired`; it must pass before any commit command.** Read
   its full output — never pipe it through `tail`, `head`, or `grep`.
   - Fix every failure without asking first, even in changes you did not
     write and even when asked to commit only what is staged.
     `make golangci-lint-fix` and `make markdownlint-fix` handle much of it.
     Stage the fixes, and name them when reporting the commit.
   - Run it again: the last run before committing must pass, with no file
     changed since. Never commit past a failing run, and never offer to.
   - This includes an amend, squash, or reword, even when an earlier run
     passed on the same tree
4. If amending, re-check every claim already in the message against the
   current diff and code, and correct or remove anything that no longer holds
5. If the branch has an open PR, re-read its body against the whole branch
   diff, and update it with `gh pr edit --body-file` when this commit changes
   behavior or makes a claim in it untrue

Then commit locally. Report the branch state and which commits are unpushed;
pushing is forbidden, per `AGENTS.md`.
