#!/bin/bash
# SessionStart hook for DDEV development in Claude Code for Web
# This hook runs at the start of each session to set up the development environment

set -e

echo "🚀 Initializing DDEV development environment for Claude Code for Web..."

# Install markdownlint-cli if not already installed
if ! command -v markdownlint &> /dev/null; then
    echo "📦 Installing markdownlint-cli..."
    npm install -g markdownlint-cli --silent 2>&1 | grep -v "npm WARN" || true
    echo "✅ markdownlint-cli installed"
else
    echo "✅ markdownlint-cli already available"
fi

# Check for other required tools
echo ""
echo "🔍 Development tools status:"
echo "  golangci-lint: $(command -v golangci-lint &> /dev/null && echo '✅ installed' || echo '❌ not found')"
echo "  markdownlint:  $(command -v markdownlint &> /dev/null && echo '✅ installed' || echo '❌ not found')"
echo "  mkdocs:        $(command -v mkdocs &> /dev/null && echo '✅ installed' || echo '⚠️  not installed (optional, will be skipped)')"
echo "  Docker:        ❌ not available in web environment"

# Environment optimizations for Claude Code for Web
echo ""
echo "⚙️  Setting environment variables for web environment..."

# Set GOTEST_SHORT to skip long-running tests that require Docker
export GOTEST_SHORT=true

# Disable Docker-dependent operations
export DDEV_NO_INSTRUMENTATION=true

# Set Go environment for faster builds
export GOCACHE="${HOME}/.cache/go-build"
export GOPATH="${HOME}/go"

echo "  GOTEST_SHORT=true (skips integration tests)"
echo "  DDEV_NO_INSTRUMENTATION=true"

echo ""
echo "📋 Important Notes:"
echo "  • Docker is NOT available in this environment"
echo "  • Integration tests requiring Docker will be skipped"
echo "  • Use 'make staticrequired' for linting (golangci-lint + markdownlint)"
echo "  • Use 'go test -short ./pkg/...' to run unit tests without Docker"
echo "  • Use 'make testpkg TESTARGS=\"-run TestName\"' for targeted package tests"
echo ""
echo "✨ Environment ready for DDEV development!"
