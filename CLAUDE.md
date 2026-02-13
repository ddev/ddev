# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DDEV is an open-source tool for running local web development environments for PHP and Node.js. It uses Docker containers to provide consistent, isolated development environments with minimal configuration.

For developer documentation, see:

- [Developer Documentation](https://docs.ddev.com/en/stable/developers/)
- [Building and Contributing](docs/content/developers/building-contributing.md)

## Key Development Commands

### Building

```bash
make                    # Build for host OS/arch. Output: .gotmp/bin/<os>_<arch>/ddev
make linux_amd64        # Cross-compile for specific platform
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

### Linting and Code Quality

These are implemented as PreToolUse and PostToolUse hooks, so should not be separately required:

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

### English Language Usage

- Never use the words `comprehensive` or `seamless`
- Avoid flowery or flattering usage in conversation, code, or comments

### Testing Philosophy

- Integration tests in `pkg/ddevapp/` test full workflows
- Documentation and docker image tests use Bats framework in `docs/tests`
- Do not commit secrets - Amplitude API keys are injected at build time

## Git Workflow

### Branch Naming

Format: `YYYYMMDD_<username>_<short_description>`

Example: `20250108_rfay_fix_networking`

### Branch Creation

```bash
git fetch upstream && git checkout -b <branch_name> upstream/main --no-track
```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

`<type>[optional scope][optional !]: <description>[, fixes #<issue>]`

Types: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `style`, `test`

Examples:

- `fix: handle container networking timeout, fixes #1234`
- `feat(pantheon): use environment variables`
- `docs: clarify mkdocs setup`

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

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
```

**Important:** Use `-F -` (read from stdin) instead of `-m "$(cat <<'EOF'...)"` to preserve all formatting.

### Creating PRs with `gh`

When creating or editing PRs with `gh pr create` or `gh pr edit`, use the same template structure from `.github/PULL_REQUEST_TEMPLATE.md` for the `--body` argument. Use a HEREDOC for the body to preserve markdown formatting:

```bash
gh pr create --title "<type>: <description>" --body "$(cat <<'EOF'
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
)"
```

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
