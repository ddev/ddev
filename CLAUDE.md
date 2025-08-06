# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Communication Style

- Use direct, concise language without unnecessary adjectives or adverbs
- Avoid flowery or marketing-style language ("tremendous", "dramatically", "revolutionary", etc.)
- Don't include flattery or excessive praise ("excellent!", "perfect!", "great job!")
- State facts and findings directly without embellishment
- Skip introductory phrases like "I'm excited to", "I'd be happy to", "Let me dive into"
- Avoid concluding with summary statements unless specifically requested
- When presenting options or analysis, lead with the core information, not commentary about it

## Project Overview

DDEV is an open-source tool for running local web development environments for PHP and Node.js. It uses Docker containers to provide consistent, isolated development environments with minimal configuration.

For comprehensive developer documentation, see:

- [Developer Documentation](https://ddev.readthedocs.io/en/stable/developers/) - Complete developer guide
- [Building and Contributing](docs/content/developers/building-contributing.md) - Local build setup and contribution workflow

## Key Development Commands

### Building

- `make` - Build for current platform

### Testing

- `make test` - Run all tests (combines testpkg and testcmd)
- `make testpkg` or `make testpkg TESTARGS="-run TestName"` - Run package tests or named test
- `make testcmd` - Run command tests  

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

- `cmd/` - Executable commands
- `pkg/` - Reusable packages
- `containers/` - Docker container definitions with Dockerfiles and configs
- `docs/` - MkDocs documentation source
- `scripts/` - Shell scripts for installation and setup
- `vendor/` - Vendored Go dependencies

### Configuration System

DDEV uses YAML configuration files:

- `.ddev/config.yaml` - Per-project configuration
- Global config stored in `~/.ddev/global_config.yaml`
- Container configs in `containers/*/` directories

## Development Notes

### Go Environment

- Uses Go modules (go.mod)
- Requires Go 1.23.0+
- Uses vendored, checked-in dependencies
- CGO is disabled by default

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

### Pull Request Titles

DDEV enforces [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format for PR titles. The format is:

`<type>[optional scope][optional !]: <description>[, fixes #<issue>][, for #<issue>]`

**Types:** `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `style`, `test`

- Use imperative, present tense ("change" not "changed")
- Don't capitalize first letter
- No period at end
- Add `, fixes #<issue>` if applicable
- Breaking changes require `!` after type

**Examples:**

- `feat: add Vite documentation section - Claude assisted with migrating blog content`
- `fix: resolve networking issues in containers - Claude helped debug connection problems`
- `docs: update CLAUDE.md with workflow guidelines - Claude suggested development improvements`
- `feat(pantheon): use environment variables`
- `fix: resolve container networking issues, fixes #1234`
- `docs: update CLAUDE.md with PR title guidelines`
- `refactor(auth): simplify user authentication flow`
- `chore(deps): bump mutagen to 0.18.1`
- `ci(pr): enforce commit message convention, fixes #5037`

### Commit Messages

**First Commit in a PR Series:** The initial commit that starts a new feature or fix should follow the structure from `.github/PULL_REQUEST_TEMPLATE.md` to provide comprehensive context. Use sections like "The Issue", "How This Commit Solves The Issue", and "Implementation Details" in the commit body, similar to the PR template format.

**Follow-up Commits:** After key Claude-initiated code changes, make commits that mention Claude and the prompt involved. This helps maintain clear attribution and context for AI-assisted development.

Examples for commit messages (without description) are provided in the "Pull Request Titles" section above.

### GitHub Issue Creation

When creating GitHub issues, always use the proper issue template format:

1. **Use the issue template structure:**
   - Preliminary checklist (with checkboxes)
   - Output of `ddev debug test` (in collapsible `<details>` section)
   - Expected Behavior
   - Actual Behavior  
   - Steps To Reproduce
   - Anything else?

2. **Include `ddev debug test` output:**

   ```bash
   ddev debug test
   ```

   Copy the **entire output** (not just the file path) into the collapsible section using this format:

   ```text
   <details><summary>Expand `ddev debug test` diagnostic information</summary>
   [triple backticks]
   [PASTE COMPLETE OUTPUT HERE]
   [triple backticks]
   </details>
   ```

3. **Create issues with gh CLI:**

   ```bash
   gh issue create --title "Title" --body-file issue.md
   ```

### Pull Request Creation

**PR Template Requirements:**

The PR template (`.github/PULL_REQUEST_TEMPLATE.md`) requires these sections:

- **The Issue:** Reference issue number with `#<issue>`
- **How This PR Solves The Issue:** Technical explanation
- **Manual Testing Instructions:** Step-by-step testing guide
- **Automated Testing Overview:** Test strategy explanation
- **Release/Deployment Notes:** Impact and considerations

**Important:** For DDEV development, commit messages should include the complete PR template content. This ensures that when the PR is created on GitHub, it will be pre-populated and won't require additional editing. Use the full template format from `.github/PULL_REQUEST_TEMPLATE.md` in your commit message body.

### Pre-Commit Workflow

**Critical Requirements Before Committing:**

1. **Run appropriate tests:**

   For targeted testing:

   ```bash
   go test -v -run TestSpecificTestName ./pkg/...
   ```

   See [Testing Documentation](https://ddev.readthedocs.io/en/stable/developers/building-contributing/#testing) for detailed testing guidance.

2. **Run static analysis:**

   ```bash
   make staticrequired
   ```

**Complete Pre-Commit Checklist:**

1. Make your changes
2. Run appropriate tests (`make test` or targeted tests)
3. Run `make staticrequired`
4. Fix any issues reported
5. Stage changes with `git add`
6. Commit with proper message format
7. Push branch and create PR
