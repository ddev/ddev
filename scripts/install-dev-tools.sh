#!/usr/bin/env bash
# Install DDEV development tools in isolated environment
# Works on macOS (brew) and Linux (apt)

set -euo pipefail

INSTALL_DIR="$HOME/.ddev-dev-tools"
PYTHON_ENV="$INSTALL_DIR/python"
NODE_ENV="$INSTALL_DIR/node"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prerequisites() {
  log_info "Checking prerequisites..."

  if ! command -v python3 >/dev/null 2>&1; then
    log_error "Python 3 is required but not found. Please install Python 3.8+."
    exit 1
  fi

  if ! command -v node >/dev/null 2>&1; then
    log_error "Node.js is required but not found. Please install Node.js 18+."
    exit 1
  fi

  log_info "✓ Python: $(python3 --version)"
  log_info "✓ Node.js: $(node --version)"
}

# Install system dependencies
install_system_deps() {
  log_info "Installing system dependencies..."

  if command -v aspell >/dev/null 2>&1; then
    log_info "Installing aspell via Homebrew..."
  else
    if command -v brew >/dev/null 2>&1; then
      log_info "Installing aspell via Homebrew..."
      brew install aspell
    elif command -v apt-get >/dev/null 2>&1; then
      log_info "Installing aspell via apt..."
      sudo apt-get update >/dev/null
      sudo apt-get install -y aspell
    else
      log_error "Neither brew nor apt-get found. Please install aspell manually."
      exit 1
    fi
  fi
}

# Setup Python environment
setup_python_env() {
  log_info "Setting up Python environment..."

  mkdir -p "$INSTALL_DIR"

  if [[ ! -f "$PYTHON_ENV/bin/activate" ]]; then
    log_info "Creating Python virtual environment..."
    python3 -m venv "$PYTHON_ENV"
  else
    log_info "✓ Python virtual environment already exists"
  fi

  # Activate and upgrade pip
  source "$PYTHON_ENV/bin/activate"
  python3 -m pip install --upgrade pip setuptools wheel >/dev/null 2>&1
}

# Install Python tools
install_python_tools() {
  log_info "Installing Python tools..."

  source "$PYTHON_ENV/bin/activate"

  cat >"$INSTALL_DIR/python-requirements.txt" <<'EOF'
# Spell checking tools
pyspelling
pymdown-extensions
EOF

  # Append docs requirements
  cat "$(dirname "$0")/../docs/requirements.txt" >>"$INSTALL_DIR/python-requirements.txt"

  log_info "Installing Python packages (this may take a moment)..."
  python3 -m pip install -r "$INSTALL_DIR/python-requirements.txt" >/dev/null
}

# Setup Node environment
setup_node_env() {
  log_info "Setting up Node.js environment..."

  mkdir -p "$NODE_ENV"
  export NPM_CONFIG_PREFIX="$NODE_ENV"
  export npm_config_update_notifier=false
  export npm_config_fund=false
}

# Install Node tools
install_node_tools() {
  log_info "Installing Node.js tools..."

  export NPM_CONFIG_PREFIX="$NODE_ENV"
  export npm_config_update_notifier=false
  export npm_config_fund=false

  npm install -g \
    markdownlint-cli \
    @umbrelladocs/linkspector \
    textlint \
    textlint-filter-rule-comments \
    textlint-rule-no-todo \
    textlint-rule-stop-words \
    textlint-rule-terminology >/dev/null
}

# Verify installation
#
# This deliberately does NOT prepend the install directories to PATH. The
# previous version did, then looked for the tools it had just installed, so it
# always succeeded while saying nothing about what the user's shell will run.
# On a machine with a Homebrew pyspelling ahead of ours on PATH it reported a
# clean install, and `pyspelling --config .spellcheck.yml` then failed with
# ModuleNotFoundError: No module named 'pymdownx'.
#
# So check two things the old version could not: which copy of each tool the
# current PATH resolves, and whether that copy actually runs.
verify_installation() {
  log_info "Verifying installation..."

  local failures=0 shadowed=0

  check_tool() {
    local tool="$1" expected_dir="$2"
    shift 2
    local resolved
    resolved="$(command -v "${tool}" 2>/dev/null || true)"

    if [[ -z "${resolved}" ]]; then
      if [[ -x "${expected_dir}/${tool}" ]]; then
        log_warn "${tool} is installed but not on your PATH (${expected_dir}/${tool})"
      else
        log_error "${tool} was not installed and is not on your PATH"
        failures=$((failures + 1))
      fi
      return
    fi

    if [[ "${resolved}" != "${expected_dir}/"* && "${expected_dir}" != "system" ]]; then
      log_warn "${tool} resolves to ${resolved}"
      log_warn "  not the copy installed here (${expected_dir}/${tool})"
      shadowed=$((shadowed + 1))
    fi

    # Presence is not enough: a tool can be on PATH and still be unusable.
    if ! "$@" >/dev/null 2>&1; then
      log_error "${tool} (${resolved}) is on PATH but failed to run: $*"
      failures=$((failures + 1))
      return
    fi

    log_info "✓ ${tool} (${resolved})"
  }

  # pyspelling only breaks when it loads a config that uses pymdown-extensions,
  # so --version would not catch the failure this is here to detect. Run it
  # against a throwaway config instead of the repo's, which would spellcheck
  # every document and could fail for reasons unrelated to the install.
  local probe; probe="$(mktemp -d)"
  printf 'hello\n' > "${probe}/probe.md"
  cat > "${probe}/probe.yml" <<EOF
matrix:
- name: probe
  aspell:
    lang: en
  pipeline:
  - pyspelling.filters.markdown:
      markdown_extensions:
      - pymdownx.superfences:
  sources:
  - '${probe}/probe.md'
EOF

  check_tool pyspelling   "$PYTHON_ENV/bin" pyspelling --config "${probe}/probe.yml"
  check_tool zensical     "$PYTHON_ENV/bin" zensical --version
  check_tool markdownlint "$NODE_ENV/bin"   markdownlint --version
  check_tool textlint     "$NODE_ENV/bin"   textlint --version
  check_tool linkspector  "$NODE_ENV/bin"   linkspector --version
  check_tool aspell       system            aspell --version
  rm -rf "${probe}"

  if [[ ${shadowed} -gt 0 ]]; then
    echo
    log_warn "${shadowed} tool(s) resolve to a different copy than the one installed here."
    log_warn "That copy may not have the dependencies these checks need."
    log_warn "make targets are unaffected: the Makefile puts ${INSTALL_DIR} first itself."
    log_warn "For your shell, put it first too:"
    log_warn "  export PATH=\"$PYTHON_ENV/bin:$NODE_ENV/bin:\$PATH\""
  fi

  if [[ ${failures} -gt 0 ]]; then
    log_error "${failures} tool(s) missing or not working"
    return 1
  fi
}

# Main installation
main() {
  log_info "Installing DDEV development tools to $INSTALL_DIR"

  check_prerequisites
  install_system_deps
  setup_python_env
  install_python_tools
  setup_node_env
  install_node_tools
  verify_installation

  echo
  log_info "Installation complete!"
  echo
  echo "The tools are now available:"
  echo "• To make: always, the Makefile puts these directories on PATH itself"
  echo "• In DDEV projects: added to PATH via .envrc, if you use direnv"
  echo "• In your shell: add to your profile (.bashrc/.bash_profile/.zshrc):"
  echo "  export PATH=\"$PYTHON_ENV/bin:$NODE_ENV/bin:\$PATH\""
}

main "$@"
