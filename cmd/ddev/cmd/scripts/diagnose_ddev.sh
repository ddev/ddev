#!/usr/bin/env bash

#ddev-generated
# diagnose_ddev.sh - User-friendly DDEV diagnostics
# This script provides concise, actionable diagnostics for DDEV issues
# For comprehensive output for issue reports, use 'ddev debug test'

set -o pipefail

# Color codes
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  BLUE='\033[0;34m'
  BOLD='\033[1m'
  NC='\033[0m' # No Color
else
  RED=''
  GREEN=''
  YELLOW=''
  BLUE=''
  BOLD=''
  NC=''
fi

ISSUES_FOUND=0
WARNINGS_FOUND=0

# Helper functions
success() {
  printf "${GREEN}✓${NC} %s\n" "$1"
}

fail() {
  printf "${RED}✗${NC} %s\n" "$1"
  ((ISSUES_FOUND++))
}

warn() {
  printf "${YELLOW}⚠${NC} %s\n" "$1"
  ((WARNINGS_FOUND++))
}

info() {
  printf "${BLUE}ℹ${NC} %s\n" "$1"
}

header() {
  printf "\n${BOLD}%s${NC}\n" "$1"
  printf '%*s\n' "${#1}" '' | tr ' ' '='
}

suggestion() {
  printf "  ${BLUE}→${NC} %s\n" "$1"
}

# Get distro info
get_distro_info() {
  if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    if [ -n "${PRETTY_NAME:-}" ]; then
      echo "$PRETTY_NAME"
    else
      echo "${ID:-unknown} ${VERSION_ID:-}"
    fi
  elif command -v lsb_release >/dev/null 2>&1; then
    lsb_release -ds
  else
    uname -sr
  fi
}

# Function to get default login shell
get_default_shell() {
  if command -v getent >/dev/null 2>&1; then
    getent passwd "$USER" | cut -d: -f7
  else
    echo "${SHELL:-unknown}"
  fi
}

# Check if project is in home directory
if [[ ${PWD} != ${HOME}* ]]; then
  printf "\n${YELLOW}WARNING:${NC} Project should usually be in a subdirectory of your home directory.\n"
  printf "Instead it's in: ${PWD}\n"
  printf "See: https://docs.ddev.com/en/stable/users/usage/troubleshooting/#project-location\n\n"
fi

header "DDEV Diagnostic Report"

# Environment Information
header "Environment"
if command -v ddev >/dev/null 2>&1; then
  ddev_version=$(ddev version -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw."DDEV version"' 2>/dev/null || echo "unknown")
  docker_platform=$(ddev version -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw."docker-platform"' 2>/dev/null || echo "unknown")
  docker_version=$(ddev version -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.docker' 2>/dev/null || echo "unknown")

  info "DDEV version: ${ddev_version}"
  info "OS: $(uname -s) $(uname -m)"
  if [ "${OSTYPE%-*}" = "linux" ]; then
    info "Distro: $(get_distro_info)"
  fi
  info "Shell: $(get_default_shell)"
  info "Docker provider: ${docker_platform}"
  info "Docker version: ${docker_version}"
else
  fail "ddev command not found in PATH"
fi

# Check if we're in a DDEV project
if ! ddev describe >/dev/null 2>&1; then
  echo
  warn "Not in a DDEV project directory"
  info "For best results, run this command in an existing DDEV project"
  echo
fi

# Docker Environment Checks
header "Docker Environment"

# Check Docker is running
if ! docker ps >/dev/null 2>&1; then
  fail "Docker is not running or not accessible"
  suggestion "Start your Docker provider (Docker Desktop, Colima, OrbStack, etc.)"
  suggestion "Check: https://docs.ddev.com/en/stable/users/install/docker-installation/"
else
  success "Docker is running"

  # Check Docker disk space
  disk_usage=$(docker run --rm ddev/ddev-utilities df -h // 2>/dev/null | awk 'NR==2 {print $5}' | tr -d '%')
  if [ -n "$disk_usage" ] && [ "$disk_usage" -gt 90 ]; then
    warn "Docker disk usage is ${disk_usage}%"
    suggestion "Run: docker system prune to free up space"
    suggestion "Or: ddev clean --all"
  elif [ -n "$disk_usage" ] && [ "$disk_usage" -gt 80 ]; then
    warn "Docker disk usage is ${disk_usage}%"
    suggestion "Consider cleaning up: docker system prune"
  else
    success "Docker disk space: ${disk_usage}% used"
  fi

  # Run dockercheck
  dockercheck_output=$(ddev utility dockercheck 2>&1)
  dockercheck_exit=$?

  if [ $dockercheck_exit -eq 0 ]; then
    # Parse dockercheck output for key info
    if echo "$dockercheck_output" | grep -q "Able to run simple container"; then
      success "Can run containers with volume mounts"
    fi
    if echo "$dockercheck_output" | grep -q "Able to use internet inside container"; then
      success "Internet access from containers"
    else
      fail "No internet access from containers"
      suggestion "Check firewall and VPN settings"
      suggestion "See: https://docs.ddev.com/en/stable/users/usage/troubleshooting/#network-issues"
    fi
    if echo "$dockercheck_output" | grep -q "docker buildx is working correctly"; then
      success "Docker buildx working"
    fi
    if echo "$dockercheck_output" | grep -q "Docker authentication is configured correctly"; then
      success "Docker authentication configured"
    fi
  else
    fail "Docker environment has issues"
    suggestion "Run 'ddev utility dockercheck' for details"
  fi
fi

# Network Checks
header "Network Connectivity"

# Check internet access from host
if curl --connect-timeout 5 --max-time 10 -sfI https://www.google.com >/dev/null 2>&1; then
  success "Internet accessible from host"
else
  fail "No internet access from host"
  suggestion "Check your network connection"
fi

# Check DNS resolution for *.ddev.site
if command -v ping >/dev/null; then
  if ping -c 1 -W 2 test.ddev.site >/dev/null 2>&1; then
    success "DNS resolves *.ddev.site to 127.0.0.1"
  else
    warn "Cannot resolve *.ddev.site"
    suggestion "See: https://docs.ddev.com/en/stable/users/usage/networking/#restrictive-dns-servers"
  fi
fi

# Check for proxy settings
if [ -n "${HTTP_PROXY:-}${http_proxy:-}${HTTPS_PROXY:-}${https_proxy:-}" ]; then
  warn "Proxy environment variables detected"
  info "HTTP_PROXY=${HTTP_PROXY:-${http_proxy:-}}"
  info "HTTPS_PROXY=${HTTPS_PROXY:-${https_proxy:-}}"
  suggestion "Proxies can interfere with DDEV. Consider NO_PROXY settings."
fi

# mkcert check
header "HTTPS/mkcert"
if command -v mkcert >/dev/null 2>&1; then
  success "mkcert is installed: $(mkcert -version 2>&1)"
  caroot=$(mkcert -CAROOT 2>/dev/null)
  if [ -f "$caroot/rootCA.pem" ]; then
    success "mkcert CA certificates exist"
  else
    warn "mkcert CA certificates not found"
    suggestion "Run: mkcert -install"
  fi
else
  warn "mkcert not found (HTTPS will use self-signed certificates)"
  suggestion "Install mkcert: https://docs.ddev.com/en/stable/users/install/ddev-installation/#macos"
fi

# Project-specific checks (if in a project)
if ddev describe >/dev/null 2>&1; then
  header "Current Project"

  project_name=$(ddev describe -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.name' 2>/dev/null || echo "unknown")
  project_status=$(ddev describe -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.status' 2>/dev/null || echo "unknown")
  project_type=$(ddev describe -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.type' 2>/dev/null || echo "unknown")

  info "Name: ${project_name}"
  info "Type: ${project_type}"
  info "Status: ${project_status}"

  # Check for customizations
  if [ -d .ddev ]; then
    custom_files=$(grep -rL "#ddev-generated" .ddev/docker-compose.*.yaml .ddev/php .ddev/nginx* .ddev/*-build .ddev/apache .ddev/mysql .ddev/postgres .ddev/.env 2>/dev/null | grep -v '\.example$')
    if [ -n "$custom_files" ]; then
      custom_count=$(echo "$custom_files" | wc -l | tr -d ' ')
    else
      custom_count=0
    fi
    if [ "$custom_count" -gt 0 ]; then
      warn "Found ${custom_count} customized configuration file(s):"
      while IFS= read -r file; do
        [ -n "$file" ] && info "  - ${file#./}"
      done <<< "$custom_files"
      suggestion "Customizations can cause issues. Try temporarily removing them for testing."
    else
      success "No custom configurations detected"
    fi

    # Check for add-ons
    addons=$(ddev add-on list --installed -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw[].Name' 2>/dev/null)
    if [ -n "$addons" ]; then
      addon_count=$(echo "$addons" | wc -l | tr -d ' ')
      info "Installed add-ons (${addon_count}):"
      while IFS= read -r addon; do
        [ -n "$addon" ] && info "  - ${addon}"
      done <<< "$addons"
    fi
  fi

  # Test if project is running
  if [ "$project_status" = "running" ]; then
    # Quick health check
    if ddev exec true >/dev/null 2>&1; then
      success "Project containers are responsive"

      # Test HTTP access
      http_url=$(ddev describe -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.httpURLs[0]' 2>/dev/null)
      if [ -n "$http_url" ] && [ "$http_url" != "null" ]; then
        http_response=$(curl --connect-timeout 5 --max-time 10 -sI "$http_url" 2>&1 | head -1)
        http_status=$(echo "$http_response" | grep -oE 'HTTP/[0-9.]+ [0-9]+' | awk '{print $2}')

        if [ -n "$http_status" ] && [ "$http_status" = "200" ]; then
          success "HTTP access working: ${http_url}"
        elif [ -n "$http_status" ]; then
          warn "HTTP returned status ${http_status} for ${http_url}"
          info "This may be normal if your project is not fully configured"
          suggestion "If unexpected, check: ddev logs"
        else
          warn "Cannot connect to project via HTTP: ${http_url}"
          suggestion "Check router: ddev logs -s router"
          suggestion "Check web container: ddev logs"
        fi
      fi
    else
      fail "Project containers not responsive"
      suggestion "Try: ddev restart"
    fi
  elif [ "$project_status" = "stopped" ]; then
    info "Project is stopped (run 'ddev start' to start it)"
  fi
fi

# Quick test project creation (optional, only if explicitly requested or there are issues)
if [ "${DDEV_DIAGNOSE_FULL:-}" = "true" ]; then
  header "Test Project Creation"

  PROJECT_NAME=ddev-diagnose-test-$$
  PROJECT_DIR=$(mktemp -d)

  info "Creating test project in ${PROJECT_DIR}..."

  cd "$PROJECT_DIR" || exit 1
  mkdir -p web
  echo "<?php phpinfo();" > web/index.php

  if ddev config --project-name="${PROJECT_NAME}" --project-type=php --docroot=web --disable-upload-dirs-warning >/dev/null 2>&1; then
    success "Test project configured"

    if timeout 120 ddev start -y >/dev/null 2>&1; then
      success "Test project started successfully"

      # Quick HTTP test
      test_url=$(ddev describe -j 2>/dev/null | docker run --rm -i ddev/ddev-utilities jq -r '.raw.httpURLs[0]' 2>/dev/null)
      if curl --connect-timeout 5 --max-time 10 -sf "$test_url" | grep -q "PHP Version" 2>/dev/null; then
        success "Test project HTTP access working"
      else
        fail "Test project HTTP access failed"
      fi

      ddev delete -Oy >/dev/null 2>&1
    else
      fail "Test project failed to start"
      suggestion "Check: ddev logs"
    fi
  else
    fail "Test project configuration failed"
  fi

  cd - >/dev/null 2>&1 || true
  rm -rf "$PROJECT_DIR"
fi

# Summary
header "Summary"

if [ $ISSUES_FOUND -eq 0 ] && [ $WARNINGS_FOUND -eq 0 ]; then
  echo
  success "${GREEN}${BOLD}All checks passed!${NC} DDEV should work correctly."
  echo
elif [ $ISSUES_FOUND -eq 0 ]; then
  echo
  warn "${YELLOW}${BOLD}${WARNINGS_FOUND} warning(s) found.${NC} DDEV should work but check warnings above."
  echo
else
  echo
  fail "${RED}${BOLD}${ISSUES_FOUND} issue(s) and ${WARNINGS_FOUND} warning(s) found.${NC}"
  echo
  printf "${BOLD}Next steps:${NC}\n"
  suggestion "Review the issues and suggestions above"
  suggestion "Check troubleshooting docs: https://docs.ddev.com/en/stable/users/usage/troubleshooting/"
  suggestion "For comprehensive diagnostics: ddev debug test"
  suggestion "Get help on Discord: https://discord.gg/5wjP76mBJD"
  echo
  exit 1
fi