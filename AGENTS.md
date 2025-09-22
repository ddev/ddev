# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## Communication Style

- Use direct, concise language without unnecessary adjectives or adverbs
- Avoid flowery or marketing-style language ("tremendous", "dramatically", "revolutionary", "working perfectly", etc.)
- Don't include flattery or excessive praise ("excellent!", "perfect!", "great job!")
- State facts and findings directly without embellishment
- Skip introductory phrases like "I'm excited to", "I'd be happy to", "Let me dive into"
- Avoid concluding with summary statements unless specifically requested
- When presenting options or analysis, lead with the core information, not commentary about it

### AI Language Guidelines

- Avoid words that reveal AI writing: "Comprehensive", "works perfectly", "You're absolutely right"
- Don't say "perfect" in response to actions
- Don't claim results are "ready for production use" without verification

## Project Overview

DDEV is an open-source tool for running local web development environments for PHP and Node.js. It uses Docker containers to provide consistent, isolated development environments with minimal configuration.

For comprehensive developer documentation, see:

- [Developer Documentation](https://docs.ddev.com/en/stable/developers/) - Complete developer guide
- [Building and Contributing](docs/content/developers/building-contributing.md) - Local build setup and contribution workflow

## Key Development Commands

### Building

- `make` - Build for host OS/arch. Output: `.gotmp/bin/<os>_<arch>/ddev`
- `make clean` - Remove build artifacts

### Testing

- `go test -v ./pkg/[package]` - Test specific package (5-30 seconds)
- `make testpkg TESTARGS="-run TestName"` - Run subset of package tests matching regex (30-120 seconds)
- `make testcmd TESTARGS="-run TestName"` - Run command tests
- `make quickstart-test` - Build then run Bats docs tests in `docs/tests`

**Testing Strategy:**

- Use subset testing with regex patterns for faster iteration
- Test specific packages when making targeted changes
- Avoid full test suite unless absolutely necessary

### Testing Environment Variables

- Set `DDEV_DEBUG=true` to see executed commands
- Set `GOTEST_SHORT=true` to limit test matrix

### Linting and Code Quality

- `make staticrequired` - Run all required static analysis (golangci-lint, markdownlint, mkdocs, pyspelling)

### Whitespace and Formatting

- **Never add trailing whitespace** - Blank lines must be completely empty (no spaces or tabs)
- Match existing indentation style exactly (spaces vs tabs, indentation depth)
- Preserve the file's existing line ending style
- Run linting tools to catch whitespace issues before committing

### Documentation

- `make staticrequired` - after changing docs

## Architecture

### Core Components

**Main Binaries:**

- `cmd/ddev/` - Main DDEV CLI application
- `cmd/ddev-hostname/` - Hostname management utility

**Core Packages:**

- `pkg/ddevapp/` - Core application logic, project management, Docker container orchestration
- `pkg/dockerutil/` - Docker container utilities and Docker Compose management
- `pkg/globalconfig/` - Global DDEV configuration management
- `pkg/fileutil/` - File system utilities
- `pkg/netutil/` - Network utilities
- `pkg/util/` - General utilities

**Container Definitions:**

- `containers/ddev-webserver/` - Web server container (Apache/Nginx + PHP)
- `containers/ddev-dbserver/` - Database server container (MySQL/MariaDB/PostgreSQL)
- `containers/ddev-nginx-proxy-router/` - Nginx reverse proxy router
- `containers/ddev-traefik-router/` - Traefik reverse proxy router
- `containers/ddev-ssh-agent/` - SSH agent container

### Project Structure

The codebase follows standard Go project structure:

- `cmd/` - CLI entrypoints (`ddev`, `ddev-hostname`)
- `pkg/` - Go packages (core app logic, Docker integration, config, utilities)
- `containers/` - Container images and Dockerfiles used by DDEV
- `docs/` - MkDocs documentation source; `docs/tests` holds Bats tests
- `scripts/` - Helper scripts (installers, tooling)
- `testing/` - Performance/auxiliary test scripts
- `vendor/` - Vendored Go dependencies

### Configuration System

DDEV uses YAML configuration files:

- `.ddev/config.yaml` - Per-project configuration
- Global config stored in `~/.ddev/global_config.yaml`
- Container configs in `containers/*/` directories

## Development Notes

### Go Environment

- Language: Go (modules + vendored deps). Use Go 1.23+
- Uses Go modules (go.mod)
- Requires Go 1.23.0+
- Uses vendored, checked-in dependencies
- CGO is disabled by default

### Coding Style & Naming Conventions

- Formatting: `gofmt` (enforced via golangci-lint). No trailing whitespace
- Linters: configured in `.golangci.yml` (errcheck, govet, revive, staticcheck, whitespace)
- Naming: packages lower-case short names; exported identifiers `CamelCase`; tests `*_test.go` with `TestXxx`

### Testing Philosophy

- Tests are organized by package
- Integration tests in `pkg/ddevapp/` test full workflows
- Container tests validate Docker functionality
- Documentation tests use bats framework

### Docker Integration

- Heavy use of docker-compose for orchestration
- Custom container images built from `containers/` directory
- Network management for inter-container communication
- Volume management for persistent data

### Code Quality

- Uses golangci-lint with specific rules in `.golangci.yml`
- Static analysis is required for CI
- Markdown linting for documentation
- Spell checking on documentation
- **Always run `make staticrequired` before committing changes** to ensure code quality standards

### Security & Configuration Tips

- Do not commit secrets. Amplitude API keys are injected at build; never hardcode them
- Docker must be installed and able to access your home directory for tests
- Before committing, run `make staticrequired` to catch style and docs issues

### Code Formatting Rules for Claude Code

- **After editing markdown files**: Run `markdownlint --fix $FILE && make markdownlint` to auto-fix and validate markdown formatting
- **After editing Go files**: Run `gofmt -w $FILE` to format Go code according to standard conventions
- **Before committing**: Run `make staticrequired` when user mentions "commit" to ensure all quality checks pass

### Development Environment Setup

- **Temporary files**: Use `~/tmp` for temporary directories and files
- **Command execution**: For bash commands that don't start with a script or executable, wrap with `bash -c "..."`
- **Working directories**: Additional common directories include `~/workspace/d11`, `~/workspace/pantheon-*`, `~/workspace/ddev.com`

### Troubleshooting & Environment Notes

**Prerequisites:**

- Go 1.25+ is required
- Docker must be installed and running
- PATH management is critical - include both `ddev` and `ddev-hostname` in PATH for testing

**Common Issues:**

- Some tests require network access and may fail in restricted environments
- Use `DDEV_NO_INSTRUMENTATION=true` to disable analytics during testing

**Important Notes:**

- Vendored dependencies are checked into the repository
- Always use absolute paths when working with repository files
- Focus on surgical, minimal changes that maintain compatibility

### Release Process

- Cross-platform builds for macOS, Linux, Windows (x64 and ARM64)
- Code signing for macOS and Windows binaries
- Chocolatey packaging for Windows
- Container image building and publishing

## Working with Claude Code

### Branch Naming

Use descriptive branch names that include:

- Date in YYYYMMDD format
- Your GitHub username
- Brief description of the work

Format: `YYYYMMDD_<username>_<short_description>`

Examples:

- `20250719_rfay_vite_docs`
- `20250719_username_fix_networking`
- `20250719_contributor_update_tests`

**Branch Creation Strategy:**

The recommended approach for creating branches is:

```bash
git fetch upstream && git checkout -b <branch_name> upstream/main --no-track
```

This method:

- Fetches latest upstream changes
- Creates branch directly from upstream/main
- Doesn't require syncing local main branch
- Uses --no-track to avoid tracking upstream/main

### Pull Request Creation

When creating pull requests for DDEV, follow the PR template structure from `.github/PULL_REQUEST_TEMPLATE.md`:

**Required Sections:**

- **The Issue:** Reference issue number with `#<issue>` and brief description
- **How This PR Solves The Issue:** Technical explanation of the solution
- **Manual Testing Instructions:** Step-by-step guide for testing changes
- **Automated Testing Overview:** Description of tests or explanation why none needed
- **Release/Deployment Notes:** Impact assessment and deployment considerations

**Commit Message Format:**

Follow Conventional Commits: `<type>[optional scope][optional !]: <description>[, fixes #<issue>]`

Examples:

- `fix: handle container networking timeout, fixes #1234`
- `docs: clarify mkdocs setup`
- `feat: add new container type support`

**For commits that will become PRs:** Include the complete PR template content in commit messages. This ensures GitHub PRs are pre-populated and don't require additional editing.

### Pre-Commit Workflow

**MANDATORY: Always run `make staticrequired` before any commit**

**Critical Requirements Before Committing:**

1. **Run appropriate tests:**

   For targeted testing:

   ```bash
   go test -v -run TestSpecificTestName ./pkg/...
   ```

   See [Testing Documentation](https://docs.ddev.com/en/stable/developers/building-contributing/#testing) for detailed testing guidance.

2. **Run static analysis (REQUIRED):**

   ```bash
   make staticrequired
   ```

   This command runs golangci-lint, markdownlint, and mkdocs. All must pass before committing.

**Complete Pre-Commit Checklist:**

1. Make your changes
2. Run appropriate tests (`make test` or targeted tests)
3. Run `make staticrequired`
4. Fix any issues reported
5. Stage changes with `git add`
6. Commit with proper message format
7. Push branch and create PR

### Validation Workflow

**Complete validation steps after making changes:**

1. **Build Validation:**

   ```bash
   make  # Wait for completion
   .gotmp/bin/<platform>/ddev --version  # Verify binary works
   ```

2. **Unit Test Validation:**

   ```bash
   go test -v ./pkg/[changed-package]  # Test your specific changes
   # Or run subset of tests matching a pattern:
   make testpkg TESTARGS="-run TestSpecificPattern"
   ```

3. **CLI Validation:**

   ```bash
   .gotmp/bin/<platform>/ddev --help  # Test CLI functionality
   .gotmp/bin/<platform>/ddev config --help  # Test command help
   ```

4. **Project Creation Validation:**

   ```bash
   # Create and configure a test project
   mkdir ~/tmp/validation-project && cd ~/tmp/validation-project
   PATH=".gotmp/bin/<platform>:$PATH" ddev config --project-type=php --docroot=web
   # Verify .ddev/config.yaml was created
   cat .ddev/config.yaml
   ```

### Claude Code Configuration

For optimal performance with DDEV development, consider these configuration patterns:

**File Exclusions** (in `.claude/settings.json`):

```json
{
  "permissions": {
    "deny": [
      "Read(./vendor/**)",
      "Read(./certfiles/**)", 
      "Read(./testing/**/artifacts/**)",
      "Read(./.git/**)",
      "Read(**/*.png)",
      "Read(**/*.jpg)",
      "Read(**/*.zip)",
      "Read(**/*.tgz)"
    ]
  }
}
```

**Common DDEV Command Allowlist**:

- `Bash(make:*)`
- `Bash(go test:*)`
- `Bash(ddev:*)`
- `Bash(gofmt:*)`
- `mcp__task-master-ai__*`

**MCP Server Configuration** (in `.mcp.json`):

- Enable `task-master-ai` for project management
- Enable `ddev` for local development operations
- Enable `github` for GitHub integration and workflow automation

**Recommended GitHub MCP Setup**:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "your_token_here"
      }
    }
  }
}
```

## Task Master AI Instructions

**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
