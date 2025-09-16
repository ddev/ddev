# DDEV Development Instructions for GitHub Copilot

Always follow these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap, Build, and Test the Repository

DDEV is a Go-based tool that uses Docker containers for local web development environments. Always build and test before making changes.

**Prerequisites:**
- Go 1.23.0+ is required
- Docker must be installed and running
- Git with upstream remotes configured

**Build Process:**
- `make` - Build for current platform. Takes ~45 seconds. NEVER CANCEL. Set timeout to 90+ seconds.
- Built binaries are placed in `.gotmp/bin/linux_amd64/` (or appropriate platform)
- Build creates both `ddev` and `ddev-hostname` binaries

**Testing:**
- `make test` - Run all tests (combines testpkg and testcmd). Takes 5-15 minutes. NEVER CANCEL. Set timeout to 30+ minutes.
- `make testpkg` - Run package tests only. Takes 2-5 minutes. NEVER CANCEL. Set timeout to 15+ minutes.
- `make testcmd` - Run command tests. Takes 10-20 minutes. NEVER CANCEL. Set timeout to 30+ minutes.
- `go test -v ./pkg/[package]` - Test specific package (fast, <30 seconds)
- `go test -v ./cmd/ddev/cmd/ -run TestSpecificTest` - Test specific command test

**Linting and Code Quality (MANDATORY before commits):**
- `make staticrequired` - Run all required static analysis (golangci-lint, markdownlint, mkdocs, pyspelling)
- `gofmt -w $FILE` - Format Go files
- `markdownlint --fix $FILE` - Fix markdown formatting

### Run DDEV Application

**Test Built DDEV Binary:**
- `.gotmp/bin/linux_amd64/ddev version` - Verify build works
- `PATH=".gotmp/bin/linux_amd64:$PATH" ddev version` - Test with proper PATH

**Create Test Project:**
```bash
mkdir /tmp/test-project && cd /tmp/test-project
PATH="/home/runner/work/ddev/ddev/.gotmp/bin/linux_amd64:$PATH" ddev config --project-type=php --docroot=web
PATH="/home/runner/work/ddev/ddev/.gotmp/bin/linux_amd64:$PATH" ddev start
```

**Note:** Container startup may fail in CI environments due to Docker limitations, but configuration commands will work.

## Validation

**ALWAYS run through these complete validation steps after making changes:**

1. **Build Validation:**
   ```bash
   make  # Wait for completion, ~45 seconds
   .gotmp/bin/linux_amd64/ddev version  # Verify binary works
   ```

2. **Unit Test Validation:**
   ```bash
   go test -v ./pkg/[changed-package]  # Test your specific changes
   ```

3. **CLI Validation:**
   ```bash
   .gotmp/bin/linux_amd64/ddev --help  # Test CLI functionality
   .gotmp/bin/linux_amd64/ddev config --help  # Test command help
   ```

4. **Project Creation Validation:**
   ```bash
   # Create and configure a test project
   mkdir /tmp/validation-project && cd /tmp/validation-project
   PATH=".gotmp/bin/linux_amd64:$PATH" ddev config --project-type=php --docroot=web
   # Verify .ddev/config.yaml was created
   cat .ddev/config.yaml
   ```

5. **Pre-Commit Validation (MANDATORY):**
   ```bash
   make staticrequired  # Must pass before any commit
   ```

## Build and Development Timing Expectations

**CRITICAL: NEVER CANCEL these operations. Wait for completion.**

- **Build (`make`):** 30-60 seconds - Set timeout to 90+ seconds
- **Full Test Suite (`make test`):** 5-15 minutes - Set timeout to 30+ minutes  
- **Package Tests (`make testpkg`):** 2-5 minutes - Set timeout to 15+ minutes
- **Command Tests (`make testcmd`):** 10-20 minutes - Set timeout to 30+ minutes
- **Static Analysis (`make staticrequired`):** 1-5 minutes - Set timeout to 10+ minutes
- **Single Package Test:** 5-30 seconds - Set timeout to 60+ seconds

## Common Development Tasks

### Repository Structure
- `cmd/` - Executable commands (`ddev`, `ddev-hostname`)
- `pkg/` - Reusable packages (core application logic)
- `containers/` - Docker container definitions
- `docs/` - MkDocs documentation source
- `scripts/` - Installation and setup scripts
- `vendor/` - Vendored Go dependencies (checked in)

### Key Packages
- `pkg/ddevapp/` - Core application logic and Docker orchestration
- `pkg/dockerutil/` - Docker utilities and docker-compose management  
- `pkg/globalconfig/` - Global configuration management
- `pkg/fileutil/` - File system utilities
- `pkg/config/` - Project configuration management

### Configuration Files
- `.ddev/config.yaml` - Per-project configuration
- `~/.config/ddev/global_config.yaml` - Global configuration
- `containers/*/` - Container-specific configurations

### Working with Tests
- Unit tests are in `pkg/*/` directories
- Integration tests are in `cmd/ddev/cmd/*_test.go`
- Documentation tests use bats framework in `docs/tests/`
- Use `-run TestName` to run specific tests
- Always test your changes with relevant test suites

### Container Development
DDEV heavily uses Docker for development environments:
- Web server containers (Apache/Nginx + PHP)
- Database containers (MySQL/MariaDB/PostgreSQL)  
- Proxy routers (Nginx/Traefik)
- SSH agent containers

Container images are built from `containers/` directory and managed via docker-compose.

### Code Quality Standards
- **No trailing whitespace** - Blank lines must be completely empty
- **Consistent indentation** - Match existing style (spaces vs tabs)
- **Go formatting** - Use `gofmt -w $FILE` for all Go files
- **Markdown linting** - Use `markdownlint --fix $FILE` for documentation
- **Static analysis** - Always run `make staticrequired` before commits

## Debugging and Troubleshooting

**Common Build Issues:**
- Ensure Go 1.23.0+ is installed
- Verify Docker is running
- Check that all Git remotes are configured (needed for version detection)

**Common Test Issues:**
- Some tests require network access and may fail in restricted environments
- Container tests may timeout in CI environments (this is expected)
- Use `DDEV_NO_INSTRUMENTATION=true` to disable analytics during testing

**Container Issues:**
- Health check timeouts are common in CI environments
- Use `ddev logs -s web` to debug container startup issues
- Container image pulls can take several minutes on first run

## Development Workflow

1. **Make your changes** to Go files, documentation, or container configurations
2. **Build and test immediately:** `make && go test -v ./pkg/[changed-package]`
3. **Validate CLI functionality:** Test relevant ddev commands manually
4. **Run static analysis:** `make staticrequired` (MANDATORY)
5. **Create test project** if needed to validate end-to-end functionality
6. **Commit changes** only after all validation passes

Always build, test, and exercise your changes manually. Simply verifying compilation is not sufficient - run actual DDEV commands and validate they behave correctly.

## Important Notes

- **CGO is disabled** by default in builds
- **Vendored dependencies** are checked into the repository
- **Docker images** are pre-built and published, but can be built locally
- **Network timeouts** are common in CI environments - this is expected
- **Container health checks** may fail in restricted Docker environments
- **Always use absolute paths** when working with files in the repository
- **PATH management** is critical - include both `ddev` and `ddev-hostname` in PATH for testing

This codebase is mature and well-tested. Focus on surgical, minimal changes that maintain compatibility with existing functionality.