# AGENTS.md

Guidance for AI agents working on the DDEV core codebase.

DDEV runs local web development environments for PHP and Node.js in Docker
containers. For developer documentation see
[Building and Contributing](docs/content/developers/building-contributing.md),
rendered at [docs.ddev.com](https://docs.ddev.com/en/stable/developers/).

## Where the rest of the guidance lives

Read the file that matches what you are touching. Claude Code loads the rules
on its own; other agents should open them directly.

| Working on | Read |
| --- | --- |
| Go code | `.claude/rules/go.md` |
| Docs or any Markdown | `.claude/rules/docs.md` |
| Apache/nginx config or templates | `.claude/rules/webserver-config.md` |
| `pkg/tui` (`ddev tui`) | `.claude/rules/tui.md` |
| A commit, PR, or issue | `.claude/skills/ddev-commit/SKILL.md` |

Fetch GitHub files through `raw.githubusercontent.com`, not
`github.com/.../blob/...` pages, which wrap the file in markup. DDEV's
[organization-wide patterns](https://raw.githubusercontent.com/ddev/.github/main/AGENTS.md)
mostly restate this file, which **wins where they differ**.

## Claude Code automation

`.claude/settings.json` applies only to Claude Code; other agents follow the
rules below by hand. With it:

- Every test variable in the Testing table below is preset.
- Editing a `.go` file runs `make golangci-lint-fix` on its package, which can
  change other files there. Editing a `.md` file lints and fixes that file.
- Common forms of `git push`, `docker push`, `go build`, and clearing
  `GOTEST_SHORT` are denied; the rules below still apply to any other form.
- The PATH from `.envrc`, with the built `ddev` and the docs tools, applies to
  every Bash call.

## Building

**Always use `make`, never `go build` directly.** `make` injects the version
through `-ldflags`, refreshes the image tags the binary references, and puts
both `ddev` and `ddev-hostname` in `.gotmp/bin/<os>_<arch>`. A bare `go build`
leaves a binary without those in the current directory while the one on PATH
stays stale. To check only that code compiles, run
`go test -run '^$' ./pkg/<package>`, which compiles the package and its tests
without running any.

```bash
make                 # Host OS/arch. Output: .gotmp/bin/<os>_<arch>/ddev
make linux_amd64     # Cross-compile for a specific platform
make windows_amd64   # Cross-compile + Windows installer; use to validate Windows changes
make clean           # Remove build artifacts
```

To exercise the binary you just built, put `.gotmp/bin/<os>_<arch>` first on
PATH, with both `ddev` and `ddev-hostname`. Otherwise a system-installed DDEV
shadows it and you test the released version without noticing.

## Testing

```bash
go test -v ./pkg/[package]                # One package
make testpkg TESTARGS="-run TestName"     # Subset of package tests
make testcmd TESTARGS="-run TestName"     # Command tests
make quickstart-test                      # Builds, then the Bats docs tests
```

**Run only the tests relevant to your change,** with `GOTEST_SHORT=true` and
`DDEV_NO_INSTRUMENTATION=true`. Never clear `GOTEST_SHORT`, and do not run
`go test ./...` or `make test` to verify one thing — the suite is slow and
broad, and CI covers the rest.

Integration tests in `pkg/ddevapp/` and `cmd/` start real projects and need
Docker; without it they fail rather than skip. `go test -short` does nothing
here, since no test in this repo calls `testing.Short()`.

| Variable | Purpose |
| --- | --- |
| `DDEV_DEBUG=true` | Raise ddev's log level to Debug, so the commands it issues and Docker status changes are printed |
| `GOTEST_SHORT=<any value>` | Cut the integration tests to a single test site. An integer picks which one, so `16` is drupal11; anything else uses the first. CI uses `16`, and leaves it unset only for the tests `.github/workflows/all-project-types-tests.sh` finds |
| `DDEV_RUN_TEST_ANYWAY=true` | Run tests that would skip under `GOTEST_SHORT` (`testcommon.SkipIfGotestShort`) or on a platform where they are unreliable |
| `DDEV_RUN_DOWNLOAD_DDEV_TEST`, `DDEV_TEST_SHARE_CMD`, `DDEV_TEST_OUTSIDE_HOME` | Set to `true` to opt in to tests of live downloads, sharing, and paths outside the home directory |
| `DDEV_NO_INSTRUMENTATION=true` | Disable analytics regardless of the global config setting |

Other skip conditions remain, so check the selected test's prerequisites.

## Linting

**`make staticrequired` must pass before every commit, including an amend.**
It runs golangci-lint, markdownlint, zensical, and an image-tag check. Read its
full output, fix what it reports, and rerun it; the commit skill has the
details.
`make golangci-lint-fix` and `make markdownlint-fix` auto-fix what those two
only report. `scripts/install-dev-tools.sh` installs the docs tooling; `make`
finds it without direnv.

IDE diagnostics from gopls can be stale — trust `make`/`go test` instead.

## Architecture

Most of the layout is self-evident from the tree; these are the parts that
are not:

- `pkg/ddevapp/` — core project logic and Docker orchestration. The `DdevApp`
  struct is a project's configuration.
- `pkg/versionconstants/` — version info and Docker image tags. **The image
  tags here are generated; never hand-edit them.** `make` runs
  `autotag-images`, which rebuilds any image whose content changed and rewrites
  its tag. `make retag-images` does the rewrite without building, and
  `make staticrequired` fails when `containers/` changed but the tags did not.
- `containers/` — the images DDEV builds and ships.
- Configuration is `.ddev/config.yaml` per project and
  `~/.ddev/global_config.yaml` globally.

## Coding style

- **Never leave trailing whitespace.** Blank lines must be completely empty.
- Match the file's existing indentation and line-ending style.
- **Prefer `require` over `assert`** in tests.
- Test the host OS with `nodeps.IsLinux()`, `nodeps.IsMacOS()`,
  `nodeps.IsWindows()`, and `nodeps.IsWSL2()` rather than comparing
  `runtime.GOOS` directly. `IsLinux()` is also true inside WSL2.
- Make surgical, minimal changes that maintain compatibility.
- Never commit secrets. Amplitude API keys are injected at build time.

## Comments

A comment says what the code does not: why, not what. **At most three lines
inside a function or block, and eight for a file header or exported doc
comment**; longer reasoning belongs in the commit message.

- Do not restate the code or the function name, or explain standard language
  or tool behavior
- Explain a thing once, not at every call site, and not again when a linked
  issue or commit message already does
- Describe what is true now, not what changed: "moved from X" goes stale

Reread every comment you wrote before finishing, and cut what breaks these
rules.

## English language usage

These rules apply everywhere: conversation, commit messages, PR text, code, and
comments.

**Never use any of these, in any form:** `comprehensive`, `seamless`,
`genuine`, `genuinely`, `honest`, `honestly`, `truly`, `really` (as an
intensifier), `perfect`, `perfectly`, `robust`, `powerful`, `effortless`,
`production-ready`, `delve`, `elevate`, `unleash`, `to be honest`, `in all
honesty`, `for real`. They assert importance instead of demonstrating it:
delete the word, and if that changes the meaning, the claim needed evidence.

- Do not compliment, flatter, or validate the reader (`You're absolutely
  right`, `Great question`, `Perfect`, `Excellent`). Lead with the substance.
- Report results plainly. If something failed, was skipped, or is unverified,
  say so.

## Git workflow

**Commit locally, never publish.** Committing, branching, and amending local
commits are expected. **Never run `git push` or `docker push`**, even when an
open PR, a CI run, or a `raw.githubusercontent.com` URL waits on it, and do not
offer to. Finish at the commit, report which commits are unpushed, and say when
a next step needs the maintainer to push.

When creating a branch, use `YYYYMMDD_<username>_<short_description>`, for
example `20250108_rfay_fix_networking`. Create one from upstream rather than
from a possibly-stale local `main`, and compare against the same base:

```bash
git fetch upstream && git checkout -b <branch_name> upstream/main --no-track
git diff upstream/main...HEAD
```

If there is no `upstream` remote, fetch `origin` and use `origin/main`.
Keep an existing task branch and preserve the user's uncommitted changes.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
`<type>[optional scope][optional !]: <description>[, fixes #<issue>]`, where
type is one of `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`,
`refactor`, `style`, or `test`. Read the commit skill before writing a commit
message, PR, or issue.

## Environment

Use `~/tmp` for temporary directories and test projects. Docker must be
installed and able to access your home directory for the integration tests.
