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

**Line endings.** Windows tools can produce CRLF. When removing line endings
from command output or text files, handle both `\r` and `\n`, for example
`strings.TrimRight(s, "\r\n")` for trailing line endings. Preserve other
whitespace and file contents when they are significant. Avoid offsets or
substring lengths that assume a fixed line-ending width.

When a change builds a host path or parses text by offset, check it against
both rules before moving on.

## Comments

The rules and the length budget in `AGENTS.md` apply here unchanged. Two Go
specifics:

Blank lines inside a doc comment are the normal way to keep a long one
readable, but not a license to exceed the eight-line budget.

A test's name and assertions already say what is being tested. Comment only
what they do not show — a non-obvious setup step, or why the test skips under
some condition.
