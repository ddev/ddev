---
paths:
  - "**/*.go"
---

# Go code in DDEV

## Windows compatibility

Two mistakes account for nearly all Windows breakage here, and neither shows up
on the machine where the change is written — passing tests on macOS or Linux is
not evidence either way.

**`path` vs `path/filepath`.** Use `filepath` for anything touching the host
filesystem — file paths, directory listings, anything passed to `os.*`. Use
`path` only for what is always forward-slash regardless of host OS: paths inside
a Docker container, URLs, embedded and import paths. Using `path`, or a manual
`"/"` join, for a host path silently produces a wrong path on Windows.

**Line endings.** Anything written or edited on Windows, or produced by a
Windows tool, can be CRLF. When parsing command output, file contents, or
config values, trim `\r` explicitly — `strings.Trim(s, "\r\n")`, not just
`"\n"` — and never hardcode byte or rune offsets, split counts, or substring
lengths that assume a fixed line-ending width. `pkg/ddevapp/snapshot.go`,
`pkg/ddevapp/addons.go`, and `pkg/nodeps/wsl.go` have the established pattern.

When a change builds a host path or parses text by offset, check it against
both rules before moving on.

## Comments

The rules and the length budget in `.claude/rules/comments.md` apply here
unchanged. Two Go specifics:

Blank lines inside a doc comment are the normal way to keep a long one
readable, but not a license to exceed the eight-line budget.

A test's name and assertions already say what is being tested. Comment only
what they do not show — a non-obvious setup step, or why the test skips under
some condition.
