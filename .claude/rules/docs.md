---
paths:
  - "docs/**"
  - "*.md"
---

# Documentation and Markdown

## Checks

CI's `docs-check.yml` also runs these three, and none of them run as part of
`make staticrequired`:

```bash
make textlint     # Terminology and stop words, per .textlintrc; README.md and docs/ only
make pyspelling   # Spellcheck, per .spellcheck.yml
make linkspector  # External link check
```

`make staticrequired` before a commit is enough — CI catches anything these
three would flag. Reach for one directly only when debugging a hit it raised.

`.textlintrc` enforces DDEV's spellings: `DDEV` rather than `ddev` in prose,
`VS Code`, `HTTP`, `HTTPS`, and `web server` as two words. The lowercase `ddev`
is correct in a command or code span, where the linter does not look.

## Line wrapping: committed files versus GitHub bodies

Committed Markdown — this file, `AGENTS.md`, everything under `docs/` — renders
as CommonMark, where a lone newline is whitespace and a paragraph reflows to the
container width, so hand-wrapping prose to a fixed column is right here.

Issue, PR, and comment **bodies** invert that rule. See
`.claude/skills/ddev-commit/SKILL.md` before writing one.
