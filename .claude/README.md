# DDEV Claude Code Configuration

This directory contains configuration for Claude Code development with DDEV.

## Directory Structure

```
.claude/
├── settings.json          # Hook configurations
├── hooks/
│   └── session-start.sh   # SessionStart hook script
└── README.md              # This file
```

## SessionStart Hook

The SessionStart hook automatically configures the development environment when starting a Claude Code session. It adapts to both desktop (with Docker) and web (without Docker) environments.

**Configuration**: Defined in `settings.json` and triggered on session startup.

**Script**: `.claude/hooks/session-start.sh` runs automatically at the start of each session.

### What It Does

1. **Installs Required Tools**
   - Installs `markdownlint-cli` via npm (required for `make staticrequired`)

2. **Detects Docker Availability**
   - Checks if Docker is installed and running
   - Adapts environment configuration based on Docker availability

3. **Sets Environment Variables**
   - `DDEV_NO_INSTRUMENTATION=true` - Disables analytics (all environments)
   - `GOTEST_SHORT=true` - Only set when Docker is NOT available (web environment)
   - Go build cache and GOPATH configuration
   - Uses `CLAUDE_ENV_FILE` to persist variables across tool calls

4. **Provides Environment Status**
   - Shows which development tools are available
   - Displays Docker availability status
   - Shows context-appropriate usage guidelines based on environment

### Environment Detection

The hook automatically detects your environment and adapts:

**Web Environment (No Docker):**
- Sets `GOTEST_SHORT=true` to skip Docker-dependent integration tests
- Shows Docker-free testing guidance
- Recommends using CI/CD for full integration tests

**Desktop Environment (With Docker):**
- Does NOT set `GOTEST_SHORT=true` (full tests available)
- Shows full testing capabilities
- Allows running complete integration test suite locally

### Tool Availability

**All Environments:**
- ✅ Go 1.24+ toolchain
- ✅ golangci-lint (pre-installed)
- ✅ markdownlint (installed by SessionStart hook)
- ✅ npm/node for tool installation
- ✅ Full ability to build DDEV binaries

**Environment-Specific:**
- ✅ Docker - Available in desktop, not in web
- ⚠️ mkdocs - Not in web (gracefully skipped by Makefile)

### Recommended Development Workflow

For development in Claude Code for Web:

```bash
# 1. Make your code changes
vim pkg/ddevapp/something.go

# 2. Run unit tests (non-Docker)
go test -short ./pkg/ddevapp

# 3. Run linting (fully supported)
make staticrequired

# 4. Build and verify
make
.gotmp/bin/linux_amd64/ddev --version

# 5. Commit and push
git add .
git commit -m "feat: your change description"
git push
```

### What Works Without Docker

- ✅ Code editing and formatting
- ✅ Static analysis with golangci-lint
- ✅ Markdown linting
- ✅ Building DDEV binaries for all platforms
- ✅ Unit tests that don't require Docker
- ✅ Reading and analyzing code

### What Requires Docker (Skip in Web Environment)

- ❌ Integration tests in `pkg/ddevapp/*_test.go` that start projects
- ❌ Container image building and testing
- ❌ Full `make test` suite
- ❌ Testing actual DDEV commands that manage projects

### CI/CD Strategy

- **Local (Web)**: Make changes, run unit tests, run linting, commit
- **CI/CD**: Automated integration tests with Docker run on push

This separation allows productive development in the web environment while ensuring full test coverage through CI/CD.
