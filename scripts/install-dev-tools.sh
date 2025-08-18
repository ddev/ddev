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
  python -m pip install --upgrade pip setuptools wheel >/dev/null 2>&1
}

# Install Python tools
install_python_tools() {
  log_info "Installing Python tools..."

  source "$PYTHON_ENV/bin/activate"

  # Download mkdocs requirements from repository
  log_info "Fetching mkdocs requirements from DDEV repository..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL https://github.com/ddev/ddev/raw/refs/heads/main/docs/mkdocs-pip-requirements \
      -o "$INSTALL_DIR/mkdocs-requirements.txt"
  elif command -v wget >/dev/null 2>&1; then
    wget -q https://github.com/ddev/ddev/raw/refs/heads/main/docs/mkdocs-pip-requirements \
      -O "$INSTALL_DIR/mkdocs-requirements.txt"
  else
    log_error "Neither curl nor wget found. Cannot download requirements."
    exit 1
  fi

  # Create combined requirements file
  cat >"$INSTALL_DIR/python-requirements.txt" <<'EOF'
# Spell checking tools
pyspelling
pymdown-extensions
EOF

  # Append mkdocs requirements
  cat "$INSTALL_DIR/mkdocs-requirements.txt" >>"$INSTALL_DIR/python-requirements.txt"

  log_info "Installing Python packages (this may take a moment)..."
  python -m pip install -q -r "$INSTALL_DIR/python-requirements.txt"
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

  npm install -g --silent \
    markdownlint-cli \
    @umbrelladocs/linkspector \
    textlint \
    textlint-filter-rule-comments \
    textlint-rule-no-todo \
    textlint-rule-stop-words \
    textlint-rule-terminology
}

# Verify installation
verify_installation() {
  log_info "Verifying installation..."

  export PATH="$PYTHON_ENV/bin:$NODE_ENV/bin:$PATH"

  local tools=(
    "mkdocs"
    "pyspelling"
    "markdownlint"
    "textlint"
    "linkspector"
    "aspell"
  )

  local missing=()
  for tool in "${tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
      log_info "✓ $tool"
    else
      missing+=("$tool")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    log_error "Missing tools: ${missing[*]}"
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
  echo "• In DDEV projects: Automatically added to PATH via .envrc"
  echo "• Globally: Add to your shell profile (.bashrc/.bash_profile/.zshrc):"
  echo "  export PATH=\"$PYTHON_ENV/bin:$NODE_ENV/bin:\$PATH\""
}

main "$@"
