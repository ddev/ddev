#!/bin/bash
# SessionStart hook for DDEV development in Claude Code
# This hook runs at the start of each session to set up the development environment
# It adapts to both desktop (with Docker) and web (without Docker) environments

set -e

echo "🚀 Initializing DDEV development environment for Claude Code..."

# Install markdownlint-cli if not already installed (useful in all environments)
if ! command -v markdownlint &> /dev/null; then
    echo "📦 Installing markdownlint-cli..."
    npm install -g markdownlint-cli --silent 2>&1 | grep -v "npm WARN" || true
    echo "✅ markdownlint-cli installed"
else
    echo "✅ markdownlint-cli already available"
fi

# Detect Docker availability
DOCKER_AVAILABLE=false
if command -v docker &> /dev/null && docker info &> /dev/null; then
    DOCKER_AVAILABLE=true
fi

# Check for other required tools
echo ""
echo "🔍 Development tools status:"
echo "  golangci-lint: $(command -v golangci-lint &> /dev/null && echo '✅ installed' || echo '❌ not found')"
echo "  markdownlint:  $(command -v markdownlint &> /dev/null && echo '✅ installed' || echo '❌ not found')"
echo "  mkdocs:        $(command -v mkdocs &> /dev/null && echo '✅ installed' || echo '⚠️  not installed (optional, will be skipped)')"

if [ "$DOCKER_AVAILABLE" = true ]; then
    echo "  Docker:        ✅ available"
else
    echo "  Docker:        ❌ not available"
fi

# Set environment variables (persist them if CLAUDE_ENV_FILE is available)
echo ""
echo "⚙️  Setting environment variables..."

# Always disable instrumentation during development
if [ -n "$CLAUDE_ENV_FILE" ]; then
    echo "export DDEV_NO_INSTRUMENTATION=true" >> "$CLAUDE_ENV_FILE"
fi
export DDEV_NO_INSTRUMENTATION=true

# Set Go environment for faster builds
if [ -n "$CLAUDE_ENV_FILE" ]; then
    echo "export GOCACHE=\"${HOME}/.cache/go-build\"" >> "$CLAUDE_ENV_FILE"
    echo "export GOPATH=\"${HOME}/go\"" >> "$CLAUDE_ENV_FILE"
fi
export GOCACHE="${HOME}/.cache/go-build"
export GOPATH="${HOME}/go"

echo "  DDEV_NO_INSTRUMENTATION=true"

# Docker-specific environment configuration
if [ "$DOCKER_AVAILABLE" = false ]; then
    # Set GOTEST_SHORT to skip long-running tests that require Docker
    if [ -n "$CLAUDE_ENV_FILE" ]; then
        echo "export GOTEST_SHORT=true" >> "$CLAUDE_ENV_FILE"
    fi
    export GOTEST_SHORT=true
    echo "  GOTEST_SHORT=true (skips integration tests - Docker not available)"

    echo ""
    echo "📋 Environment Notes (No Docker):"
    echo "  • Docker is NOT available in this environment"
    echo "  • Integration tests requiring Docker will be skipped"
    echo "  • Use 'go test -short ./pkg/...' to run unit tests without Docker"
    echo "  • Use 'make testpkg TESTARGS=\"-run TestName\"' for targeted package tests"
    echo "  • Full integration tests will run in CI/CD"
else
    echo ""
    echo "📋 Environment Notes (Docker Available):"
    echo "  • Docker is available - full integration tests can run"
    echo "  • Use 'make test' to run the full test suite"
    echo "  • Use 'go test -v ./pkg/[package]' to test specific packages"
    echo "  • Use 'make testpkg TESTARGS=\"-run TestName\"' for targeted tests"
fi

echo ""
echo "💡 Always run 'make staticrequired' before committing (golangci-lint + markdownlint)"
echo ""
echo "✨ Environment ready for DDEV development!"
