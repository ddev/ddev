# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DDEV is an open-source tool for running local web development environments for PHP and Node.js. It uses Docker containers to provide consistent, isolated development environments with minimal configuration.

For developer documentation, see:

- [Developer Documentation](https://docs.ddev.com/en/stable/developers/)
- [Building and Contributing](docs/content/developers/building-contributing.md)

## Key Development Commands

### Building

**IMPORTANT: Always use `make` to build the DDEV binary, never `go build` directly.**

```bash
make                    # Build for host OS/arch. Output: .gotmp/bin/<os>_<arch>/ddev
make linux_amd64        # Cross-compile for specific platform
make windows_amd64      # Cross-compile + build Windows installer .exe (use this to validate Windows changes)
```

### Testing

```bash
go test -v ./pkg/[package]                    # Test specific package
make testpkg TESTARGS="-run TestName"         # Run subset of package tests
make testcmd TESTARGS="-run TestName"         # Run command tests
make quickstart-test                          # Run Bats docs tests
```

**Testing Tips:**

- Use subset testing with regex patterns for faster iteration
- Set `DDEV_DEBUG=true` to see executed commands
- Set `GOTEST_SHORT=true` to limit test matrix
- `DDEV_NO_INSTRUMENTATION=true` should always be set to disable analytics
- **When verifying a change locally, run only the tests relevant to it** (e.g. `go test -run TestName ./pkg/[package]`). Do not run the full suite (`go test ./...` or `make test`) — it is slow and broad; let CI cover the rest.

### Linting and Code Quality

**IMPORTANT: Always run `make staticrequired` before committing. A PreToolUse hook in `.claude/settings.json` runs it automatically before every `git commit`.**

**IDE diagnostics (from gopls via the VSCode extension) can be stale and should not be trusted over `make` or `go test`. If the build is clean, ignore IDE diagnostics.**

`gofmt` and `markdownlint --fix` run automatically via PostToolUse hooks after editing Go or Markdown files. Run them manually if needed:

```bash
make staticrequired                           # Run all required static analysis
gofmt -w $FILE                                # Format Go files after editing
markdownlint --fix $FILE                      # Fix markdown formatting
```

## Architecture

### Core Components

**Main Binaries** (`cmd/`):

- `cmd/ddev/` - Main CLI application using Cobra framework
- `cmd/ddev-hostname/` - Hostname management utility

**Core Packages** (`pkg/`):

- `pkg/ddevapp/` - Core application logic, project management, Docker orchestration. The `DdevApp` struct represents a DDEV project configuration.
- `pkg/dockerutil/` - Docker client utilities and Docker Compose management
- `pkg/globalconfig/` - Global DDEV configuration (`~/.ddev/global_config.yaml`)
- `pkg/versionconstants/` - Version info and Docker image tags. **Edit this when testing custom container images.**
- `pkg/fileutil/`, `pkg/netutil/`, `pkg/util/` - Utility packages

**Container Definitions** (`containers/`):

- `ddev-webserver/` - Web server (Apache/Nginx + PHP)
- `ddev-dbserver/` - Database server (MySQL/MariaDB/PostgreSQL)
- `ddev-traefik-router/` - Traefik-based router
- `ddev-ssh-agent/` - SSH agent container

### Configuration System

- `.ddev/config.yaml` - Per-project configuration
- `~/.ddev/global_config.yaml` - Global configuration

### Webserver Config Has Two Independent Copies — Keep Them In Sync

Apache and nginx site configs are baked into the `ddev-webserver` image
(`containers/ddev-webserver/ddev-webserver-base-files/etc/{apache2,nginx}/...`),
but `start.sh` replaces them wholesale from the *project's* `.ddev/` directory
when it exists:

- `.ddev/nginx_full/` → replaces `/etc/nginx/sites-enabled`
- `.ddev/apache/` → replaces `/etc/apache2/sites-enabled`

Those project files come from the `ddev` **CLI binary**, not the image: a
`//go:embed`d copy under `pkg/ddevapp/webserver_config_assets/`, written by
`GenerateWebserverConfig()` in `pkg/ddevapp/ddevapp.go`. They carry a
`#ddev-generated` marker and are refreshed on `ddev start`/`ddev config`, so they
always win over what's in the image, however often you rebuild it.

**When changing a real (non-fallback) apache or nginx site template, check
whether `pkg/ddevapp/webserver_config_assets/` has the same content and update it
too, then rebuild the `ddev` binary (`make`), not just the image.** nginx site
templates `include /etc/nginx/common.d/*.conf`, so prefer putting fixes there
(image-only, no duplication). Apache has no such include, so apache fixes usually
need both copies.

To pick up a `webserver_config_assets` change in an existing project, rebuild the
binary and run `ddev start`. A project that removed the `#ddev-generated` line
owns its file and won't be touched.

## Development Notes

### Go Environment

- **Go 1.24+** required (see `go.mod`)
- Uses vendored, checked-in dependencies (`vendor/`)
- CGO disabled by default

### Coding Style

- Formatting: `gofmt` enforced via golangci-lint
- Linters configured in `.golangci.yml`: errcheck, govet, revive, staticcheck, whitespace
- **Never add trailing whitespace** - blank lines must be completely empty
- **Prefer `require` over `assert`** in tests for all assertions
- Focus on surgical, minimal changes that maintain compatibility
- Tests should prefer `require` over `assert`

#### Comments

Keep comments short. A comment earns its place only by saying something the code
does not already say — why a directive is ordered that way, why a workaround
exists, a non-obvious consequence.

- Do not restate the code, the function name, or the config directive below it
- Do not explain standard behavior of the language, webserver, or tooling
- Do not write multi-paragraph rationale essays; one to three lines is usually enough
- Do not repeat the same explanation in several files — explain it once and point
  to that file
- Never re-describe in comments what a linked issue or commit message already covers

### English Language Usage

These rules apply everywhere: conversation, commit messages, PR text, code, and comments.

- **Forbidden words. Never use any of these, in any form, anywhere — including
  ordinary conversation, not just written deliverables:**
  `comprehensive`, `seamless`, `genuine`, `genuinely`, `honest`, `honestly`,
  `truly`, `really` (as an intensifier), `perfect`, `perfectly`, `robust`,
  `powerful`, `effortless`, `production-ready`, `delve`, `elevate`, `unleash`,
  `to be honest`, `in all honesty`, `for real`.
- These are forbidden because they assert sincerity or importance instead of
  demonstrating it. `genuinely useful` is not more useful than `useful`; a
  `genuine bug` is just a bug. Delete the word — the sentence is stronger
  without it. If deleting changes the meaning, the claim needed evidence, not
  an intensifier.
- Watch for them especially when summarizing your own work, which is where the
  urge to add emphasis is strongest.
- Do not compliment, flatter, or validate the user (`You're absolutely right`,
  `Great question`, `Perfect`, `Excellent`). Skip the preamble and answer.
- Do not open a reply by praising the request or restating how good the idea is.
  Lead with the substance.
- Report results plainly. If something failed, was skipped, or is unverified, say so;
  do not paper over it with confident-sounding language.

### Testing Philosophy

- Integration tests in `pkg/ddevapp/` test full workflows
- Documentation and docker image tests use Bats framework in `docs/tests`
- Do not commit secrets - Amplitude API keys are injected at build time

## Git Workflow

### Commit Locally, Never Push

Committing locally is fine and expected — commit whenever asked, without
hesitation. Creating branches and amending local commits is fine too.

**What is forbidden is publishing: never run `git push` or `docker push`, under
any circumstances.** This holds even when a PR is already open and waiting on
the commits, and even when something downstream (a CI run, a
`raw.githubusercontent.com` URL someone else needs) depends on the push.
Pushing is always the maintainer's action.

Do not offer to push either. Finish at the commit, report the branch state and
which commits are unpushed, and stop. If a next step genuinely requires a push,
say plainly that it will not work until the maintainer pushes, rather than
asking whether to push.

### Branch Naming

Format: `YYYYMMDD_<username>_<short_description>`

Example: `20250108_rfay_fix_networking`

### Branch Creation

```bash
git fetch upstream && git checkout -b <branch_name> upstream/main --no-track
```

### Comparing Against Upstream

When generating diffs or comparing branches for a PR, prefer `upstream/main` as the base if an `upstream` remote exists. Local `main` may be out of date. If there is no `upstream` remote, fall back to `origin/main`.

```bash
git fetch upstream 2>/dev/null || git fetch origin
git diff upstream/main...HEAD 2>/dev/null || git diff origin/main...HEAD
```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

`<type>[optional scope][optional !]: <description>[, fixes #<issue>]`

Types: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `style`, `test`

Examples:

- `fix: handle container networking timeout, fixes #1234`
- `feat(pantheon): use environment variables`
- `docs: clarify zensical setup`

**ALWAYS write the commit message to a file first and commit with `git commit -F <file>`.**
This applies to every commit, not just PR-initial commits. Never use
`git commit -m "$(cat <<'EOF' ... EOF)"` or any other inline-heredoc-in-`$(...)`
construct — it is unreliable in bash and silently fails or mangles the message
(missing EOF, stray quotes, chained `&&` breaking mid-heredoc). Write the message
with the Write tool (or `cat > /tmp/msg.txt <<'EOF' ... EOF` as its own command),
then run `git commit -F /tmp/msg.txt` as a separate command.

Keep the body short. Each template section is a few lines or bullets, written for
a reviewer who will read the diff anyway:

- Say what changed and why, once — do not narrate the whole diff file by file
- Do not repeat the same point in several sections
- Do not pad "Manual Testing Instructions" with obvious steps; give the commands
  and what to look for
- Write "None" where a section does not apply instead of inventing content

### Pull Request Template

In the initial commit for a PR, use the format in  `.github/PULL_REQUEST_TEMPLATE.md` with these required sections:

- **The Issue:** Reference issue with `#<number>`
- **How This PR Solves The Issue:** Technical explanation
- **Manual Testing Instructions:** Step-by-step testing guide
- **Automated Testing Overview:** Test coverage explanation
- **Release/Deployment Notes:** Impact assessment

### Creating Commits with PR Template

When creating the initial commit for a PR, use `git commit -F -` to read from stdin. This preserves markdown formatting including `##` headers:

```bash
cat <<'EOF' | git commit -F -
<type>: <description>

## The Issue

- Fixes #<issue_number>

[Issue description]

## How This PR Solves The Issue

[Technical explanation]

## Manual Testing Instructions

[Step-by-step testing guide]

## Automated Testing Overview

[Test coverage explanation]

## Release/Deployment Notes

[Impact assessment]

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
```

**Important:** Use `-F -` (read from stdin) instead of `-m "$(cat <<'EOF'...)"` to preserve all formatting.

### Creating PRs with `gh`

When creating or editing PRs with `gh pr create` or `gh pr edit`, use the same template structure from `.github/PULL_REQUEST_TEMPLATE.md`. **Always write the body to a temporary file and use `--body-file`** — inline HEREDOCs inside `$(...)` are unreliable in bash and will fail when the body contains single quotes or special characters.

```bash
cat > ~/tmp/pr_body.md <<'EOF'
## The Issue

- Fixes #<issue_number>

[Issue description]

## How This PR Solves The Issue

[Technical explanation]

## Manual Testing Instructions

[Step-by-step testing guide]

## Automated Testing Overview

[Test coverage explanation]

## Release/Deployment Notes

[Impact assessment]
EOF
gh pr create --title "<type>: <description>" --body-file ~/tmp/pr_body.md
```

### Avoiding Hard Line Breaks in Issue/PR/Comment Bodies

GitHub renders issue, PR, and comment bodies (`gh issue create`, `gh pr create`, `gh pr comment`, `gh issue comment`, etc.) with GFM's hard-line-break behavior: a single `\n` inside a paragraph becomes an actual `<br>`. This is different from how GitHub renders committed Markdown files (this file, docs, READMEs), which follow standard CommonMark, where a lone `\n` is just whitespace and the paragraph reflows to the container width.

Hand-wrapping prose to a fixed column width — normal, good practice for a text file or commit message body — produces a ragged, too-short-lined paragraph when posted as an issue/PR/comment body, because each wrapped line becomes its own forced line instead of reflowing.

When writing a `--body-file` for any of these commands, write each paragraph as one continuous line with no embedded newlines. Only use actual blank lines to separate paragraphs, headings, and list items. This does not apply to code blocks, tables, or files meant to be read as source.

### Pre-Commit Checklist

1. Run appropriate tests: `go test -v -run TestName ./pkg/...`
2. Run static analysis: `make staticrequired`
3. Fix any issues
4. Commit with proper message format

## Validation Workflow

```bash
# 1. Build
make
.gotmp/bin/<platform>/ddev --version

# 2. Test specific changes
go test -v ./pkg/[changed-package]

# 3. Validate CLI
.gotmp/bin/<platform>/ddev --help

# 4. Test project creation (optional)
mkdir ~/tmp/test-project && cd ~/tmp/test-project
.gotmp/bin/<platform>/ddev config --project-type=php --docroot=web
```

## Environment Notes

### Prerequisites

- Go 1.24+ installed
- Docker installed and running
- `~/tmp` available for test directories
- Include both `ddev` and `ddev-hostname` in PATH when testing

### Useful Environment Variables

| Variable                       | Purpose                |
| ------------------------------ | ---------------------- |
| `DDEV_DEBUG=true`              | Show executed commands |
| `GOTEST_SHORT=true`            | Limit test matrix      |
| `DDEV_NO_INSTRUMENTATION=true` | Disable analytics      |

### Web Environment (No Docker)

When Docker is unavailable:

- Run unit tests: `go test -short ./pkg/...`
- Run linting: `make staticrequired`
- Build: `make`
- Let CI run integration tests
