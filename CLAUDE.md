# CLAUDE.md

@AGENTS.md

## Claude Code

<!--
Maintainer note, free because block-level HTML comments are stripped before
this file reaches Claude's context: anything the harness can enforce belongs in
.claude/settings.json, not in prose here. See .claude/README.md.
-->

The harness already enforces several of the rules above, so do not restate
them as reminders or re-verify them by hand:

- `make staticrequired` runs automatically before every `git commit`.
- `git push`, `docker push`, and `go build` are denied outright.
- `GOTEST_SHORT=true` and `DDEV_NO_INSTRUMENTATION=true` are preset for every
  command. Prefix a command with `GOTEST_SHORT=` to run the full matrix.
- Editing a `.go` or `.md` file formats it automatically.
- `.gotmp/bin/<os>_<arch>` is already first on PATH, so the binary you just
  built is what runs — no need to adjust PATH yourself.

`.claude/rules/` files load themselves when you read a file they cover, so
there is no need to open them by hand. Run `/ddev-commit` before writing a
commit message, PR, or issue.
