# AGENTS.md

Guidance for AI agents working on the DDEV core codebase. This is the canonical
file: `CLAUDE.md` imports it and adds only Claude Code specifics, and
`.github/copilot-instructions.md` is a symlink to it. Edit this file, not
those.

DDEV runs local web development environments for PHP and Node.js in Docker
containers. For developer documentation see
[Building and Contributing](docs/content/developers/building-contributing.md),
rendered at [docs.ddev.com](https://docs.ddev.com/en/stable/developers/).

## Where the rest of the guidance lives

Read the file that matches what you are touching. Claude Code loads these on
its own; other agents should open them directly.

| Working on | Read |
| --- | --- |
| Go code | `.claude/rules/go.md` |
| Docs or any Markdown | `.claude/rules/docs.md` |
| Apache/nginx config or templates | `.claude/rules/webserver-config.md` |
| A commit, PR, or issue | `.claude/skills/ddev-commit/SKILL.md` |
| Any comment, in any file | `.claude/rules/comments.md` |

Fetch files from GitHub through `raw.githubusercontent.com`; a
`github.com/.../blob/...` page wraps the file in markup that costs tokens and
can be summarized instead of read. DDEV's
[organization-wide patterns](https://raw.githubusercontent.com/ddev/.github/main/AGENTS.md)
mostly restate this file, which **wins where they differ** — it is stricter
about never pushing.

## Building

**Always use `make`, never `go build` directly.**

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

**Run only the tests relevant to your change.** Do not run `go test ./...` or
`make test` to verify one thing — the suite is slow and broad, and CI covers
the rest.

Integration tests in `pkg/ddevapp/` and `cmd/` start real projects and need
Docker; without it they fail rather than skip. `go test -short` does nothing
here, since no test in this repo calls `testing.Short()`.

| Variable | Purpose |
| --- | --- |
| `DDEV_DEBUG=true` | Raise ddev's log level to Debug, so the commands it issues and Docker status changes are printed |
| `GOTEST_SHORT=<any value>` | Cut the integration tests to a single test site. An integer picks which one, so `16` is drupal11; anything else uses the first. Leave it unset for the full matrix |
| `DDEV_NO_INSTRUMENTATION=true` | Disable analytics regardless of the global config setting |

## Linting

**Run `make staticrequired` before committing.** It runs golangci-lint,
markdownlint, zensical, and an image-tag check, and all must pass.

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

## Go environment

Go 1.27 or newer, per `go.mod`. Modules with vendored dependencies checked into
`vendor/`, CGO disabled by default.

## Coding style

- `.golangci.yml` enables errcheck, govet, ineffassign, modernize, revive,
  staticcheck, and whitespace, plus `gofmt` as a formatter.
- **Never leave trailing whitespace.** Blank lines must be completely empty.
- Match the file's existing indentation and line-ending style.
- **Prefer `require` over `assert`** in tests.
- Make surgical, minimal changes that maintain compatibility.
- Never commit secrets. Amplitude API keys are injected at build time.

### Comments

A comment earns its place only by saying something the code does not — why,
not what — within a three-line budget inside a function or block (eight for a
file header or an exported doc comment); longer reasoning belongs in the
commit message. Applies in every language here. See
`.claude/rules/comments.md` for the full rule set and a worked example.

## English language usage

These rules apply everywhere: conversation, commit messages, PR text, code, and
comments.

**Never use any of these, in any form:** `comprehensive`, `seamless`,
`genuine`, `genuinely`, `honest`, `honestly`, `truly`, `really` (as an
intensifier), `perfect`, `perfectly`, `robust`, `powerful`, `effortless`,
`production-ready`, `delve`, `elevate`, `unleash`, `to be honest`, `in all
honesty`, `for real`.

They assert sincerity or importance instead of demonstrating it: `genuinely
useful` is not more useful than `useful`. Delete the word, and if that changes
the meaning, the claim needed evidence rather than an intensifier. The urge is
strongest when summarizing your own work.

- Do not compliment, flatter, or validate the reader (`You're absolutely
  right`, `Great question`, `Perfect`, `Excellent`), and do not open by praising
  the request. Lead with the substance.
- Report results plainly. If something failed, was skipped, or is unverified,
  say so.

## Git workflow

**Commit locally, never publish.** Committing whenever asked is expected, as is
creating branches and amending local commits. **Never run `git push` or
`docker push`, under any circumstances** — not when a PR is already open and
waiting on the commits, and not when a CI run or a
`raw.githubusercontent.com` URL depends on it. Do not offer to, either: finish
at the commit, report which commits are unpushed, and say plainly when a next
step needs a push, which is always the maintainer's action.

Branch names are `YYYYMMDD_<username>_<short_description>`, for example
`20250108_rfay_fix_networking`. Create one from upstream rather than from a
possibly-stale local `main`, and compare against the same base:

```bash
git fetch upstream && git checkout -b <branch_name> upstream/main --no-track
git diff upstream/main...HEAD
```

If there is no `upstream` remote, fall back to `origin/main`.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
`<type>[optional scope][optional !]: <description>[, fixes #<issue>]`, where
type is one of `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`,
`refactor`, `style`, or `test`. Message discipline, the PR and issue templates,
and how to drive `gh` are in `.claude/skills/ddev-commit/SKILL.md`. Read it
before writing any of them.

## Environment

Use `~/tmp` for temporary directories and test projects. Docker must be
installed and able to access your home directory for the integration tests.
