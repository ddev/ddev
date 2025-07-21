# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DDEV is an open-source tool for running local web development environments for PHP and Node.js. It uses Docker containers to provide consistent, isolated development environments with minimal configuration.

For comprehensive developer documentation, see:

- [Developer Documentation](https://ddev.readthedocs.io/en/stable/developers/) - Complete developer guide
- [Building and Contributing](docs/content/developers/building-contributing.md) - Local build setup and contribution workflow

## Key Development Commands

### Building

- `make build` - Build for current platform
- `make darwin_amd64` - Build for macOS Intel
- `make darwin_arm64` - Build for macOS Apple Silicon  
- `make linux_amd64` - Build for Linux x64
- `make windows_amd64` - Build for Windows x64
- `make completions` - Generate shell completions

### Testing

- `make test` - Run all tests (combines testpkg and testcmd)
- `make testpkg` - Run package tests
- `make testcmd` - Run command tests  
- `make testddevapp` - Run ddevapp package tests specifically
- `make testnotddevapp` - Run all package tests except ddevapp
- `make testfullsitesetup` - Run full site setup tests
- `make quickstart-test` - Run quickstart documentation tests using bats

### Linting and Code Quality

- `make golangci-lint` - Run Go linter (requires golangci-lint to be installed)
- `make staticrequired` - Run all required static analysis (golangci-lint, markdownlint, mkdocs, pyspelling)
- `make markdownlint` - Lint markdown files
- `make pyspelling` - Check spelling

### Documentation

- `make mkdocs` - Build documentation
- `make mkdocs-serve` - Serve docs locally for development
- See [Testing Documentation](https://ddev.readthedocs.io/en/stable/developers/testing-docs/) for docs setup

### Cleanup

- `make clean` or `make bin-clean` - Remove build artifacts

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
- Uses vendored dependencies
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

Examples:

- `20250719_rfay_vite_docs`
- `20250719_username_fix_networking`
- `20250719_contributor_update_tests`

### Commit Messages

After key Claude-initiated code changes, make a commit, and the commit message should mention Claude and the prompt involved. This helps maintain clear attribution and context for AI-assisted development.

Examples:

- `feat: add Vite documentation section - Claude assisted with migrating blog content`
- `fix: resolve networking issues in containers - Claude helped debug connection problems`
- `docs: update CLAUDE.md with workflow guidelines - Claude suggested development improvements`
