#!/usr/bin/env bash
# Install DDEV development tools in isolated environment
# Works on macOS (brew) and Linux (apt)

set -euo pipefail

INSTALL_DIR="$HOME/.ddev-dev-tools"
PYTHON_ENV="$INSTALL_DIR/python"
NODE_ENV="$INSTALL_DIR/node"
PUPPETEER_CACHE="$INSTALL_DIR/puppeteer"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DIRENV_RELOAD_NEEDED=false

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

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

install_system_deps() {
  log_info "Installing system dependencies..."

  if command -v aspell >/dev/null 2>&1; then
    log_info "✓ aspell already installed"
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

setup_python_env() {
  log_info "Setting up Python environment..."

  mkdir -p "$INSTALL_DIR"

  # A venv is bound to the interpreter that created it, so a system Python
  # upgrade leaves it dead. Rebuild rather than fail halfway through pip.
  if [[ -f "$PYTHON_ENV/bin/activate" ]] && ! "$PYTHON_ENV/bin/python3" -c '' 2>/dev/null; then
    log_warn "Python virtual environment is broken (system Python changed); recreating"
    rm -rf "$PYTHON_ENV"
  fi

  if [[ ! -f "$PYTHON_ENV/bin/activate" ]]; then
    log_info "Creating Python virtual environment..."
    python3 -m venv "$PYTHON_ENV"
  else
    log_info "✓ Python virtual environment already exists"
  fi

  # Not `source bin/activate`: that would put $PYTHON_ENV/bin on PATH for the
  # rest of this script, so verify_installation would not see the user's PATH.
  "$PYTHON_ENV/bin/python3" -m pip install --upgrade pip setuptools wheel >/dev/null 2>&1
}

install_python_tools() {
  log_info "Installing Python tools..."

  cat >"$INSTALL_DIR/python-requirements.txt" <<'EOF'
# Spell checking tools
pyspelling
pymdown-extensions
EOF
  cat "$REPO_ROOT/docs/requirements.txt" >>"$INSTALL_DIR/python-requirements.txt"

  log_info "Installing Python packages (this may take a moment)..."
  # --upgrade so re-running updates; without it pip stops at "already satisfied".
  "$PYTHON_ENV/bin/python3" -m pip install --upgrade -r "$INSTALL_DIR/python-requirements.txt" >/dev/null
}

setup_node_env() {
  log_info "Setting up Node.js environment..."

  mkdir -p "$NODE_ENV"
  export NPM_CONFIG_PREFIX="$NODE_ENV"
  export npm_config_update_notifier=false
  export npm_config_fund=false
}

install_node_tools() {
  log_info "Installing Node.js tools (this may take a moment)..."

  # puppeteer's postinstall downloads linkspector's browser, which npm 12 blocks
  # unless allowlisted. PUPPETEER_CACHE_DIR keeps it out of ~/.cache/puppeteer.
  export PUPPETEER_CACHE_DIR="$PUPPETEER_CACHE"
  npm install -g --allow-scripts=puppeteer \
    markdownlint-cli \
    @umbrelladocs/linkspector \
    textlint \
    textlint-filter-rule-comments \
    textlint-rule-no-todo \
    textlint-rule-stop-words \
    textlint-rule-terminology >/dev/null

  write_linkspector_shim
  prune_puppeteer_cache
}

puppeteer_dir() {
  find "$NODE_ENV/lib/node_modules" -type d -path '*/node_modules/puppeteer' -print -quit 2>/dev/null
}

# puppeteer adds each browser build it downloads (~650MB) instead of replacing
# the last one, so drop whatever the installed version no longer resolves to.
prune_puppeteer_cache() {
  local pd exe rel keep browser build
  pd="$(puppeteer_dir)"
  [[ -n "${pd}" ]] || return 0

  exe="$(cd "${pd}" && PUPPETEER_CACHE_DIR="$PUPPETEER_CACHE" node -e "
    import('puppeteer').then(async m => console.log(await (m.default ?? m).executablePath()))
  " 2>/dev/null)" || return 0

  rel="${exe#"$PUPPETEER_CACHE"/}"
  [[ "${rel}" != "${exe}" ]] || return 0 # resolved outside our cache; leave it alone
  rel="${rel#*/}"
  keep="${rel%%/*}"

  # chrome and chrome-headless-shell are fetched together at the same build id.
  for browser in "$PUPPETEER_CACHE"/*/; do
    for build in "${browser}"*/; do
      [[ -d "${build}" ]] || continue
      if [[ "$(basename "${build}")" == "${keep}" ]]; then continue; fi
      log_info "Removing stale browser $(basename "${browser%/}")/$(basename "${build}")"
      rm -rf "${build}"
    done
  done
}

# PUPPETEER_CACHE_DIR has to be set when linkspector runs, not only when it is
# installed, so carry it in the bin instead of in every caller's environment.
write_linkspector_shim() {
  local entry="$NODE_ENV/lib/node_modules/@umbrelladocs/linkspector/index.js"
  local bin="$NODE_ENV/bin/linkspector"

  if [[ ! -f "${entry}" ]]; then
    log_warn "linkspector entry point not found at ${entry}"
    return
  fi

  # rm first: writing to the symlink npm created would overwrite index.js.
  rm -f "${bin}"
  cat >"${bin}" <<EOF
#!/usr/bin/env bash
# Generated by ddev scripts/install-dev-tools.sh; rewritten on each install.
export PUPPETEER_CACHE_DIR="\${PUPPETEER_CACHE_DIR:-${PUPPETEER_CACHE}}"
exec node "${entry}" "\$@"
EOF
  chmod +x "${bin}"
}

# True when .envrc will put the tool directories on PATH but direnv has not
# re-evaluated yet, which is always so on a first install: .envrc took its
# "tools missing" branch before the directories existed.
direnv_will_cover_path() {
  [[ -n "${DIRENV_DIR:-}" ]] || return 1
  command -v direnv >/dev/null 2>&1 || return 1
  [[ -f "${REPO_ROOT}/.envrc" ]] || return 1
  grep -q 'ddev-dev-tools' "${REPO_ROOT}/.envrc" || return 1
  # "allowed 0" is direnv's code for allowed; a blocked .envrc will not reload.
  (cd "${REPO_ROOT}" && direnv status 2>/dev/null | grep -q '^Found RC allowed 0$')
}

verify_installation() {
  log_info "Verifying installation..."

  local failures=0 path_issues=0 direnv_covers=false
  if direnv_will_cover_path; then
    direnv_covers=true
  fi

  check_tool() {
    local tool="$1" expected_dir="$2"
    shift 2
    local resolved
    resolved="$(command -v "${tool}" 2>/dev/null || true)"

    if [[ -z "${resolved}" ]]; then
      if [[ ! -x "${expected_dir}/${tool}" ]]; then
        log_error "${tool} was not installed and is not on your PATH"
        failures=$((failures + 1))
        return
      fi
      # Run it anyway, so a broken install is not hidden behind a PATH message.
      shift # drop the tool name; call the copy we installed by full path
      if ! "${expected_dir}/${tool}" "$@" >/dev/null 2>&1; then
        log_error "${tool} (${expected_dir}/${tool}) failed to run: $*"
        failures=$((failures + 1))
        return
      fi

      if [[ "${direnv_covers}" == true ]]; then
        DIRENV_RELOAD_NEEDED=true
        log_info "✓ ${tool} (${expected_dir}/${tool}), on PATH after direnv reloads"
      else
        log_warn "${tool} is installed but not on your PATH (${expected_dir}/${tool})"
        path_issues=$((path_issues + 1))
      fi
      return
    fi

    if [[ "${resolved}" != "${expected_dir}/"* && "${expected_dir}" != "system" ]]; then
      log_warn "${tool} resolves to ${resolved}"
      log_warn "  not the copy installed here (${expected_dir}/${tool})"
      path_issues=$((path_issues + 1))
    fi

    if ! "$@" >/dev/null 2>&1; then
      log_error "${tool} (${resolved}) is on PATH but failed to run: $*"
      failures=$((failures + 1))
      return
    fi

    log_info "✓ ${tool} (${resolved})"
  }

  # pyspelling only breaks when a config pulls in pymdown-extensions, so
  # --version misses it. Probe with a throwaway config, not the repo's, which
  # could fail over document content.
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

  # linkspector launches a browser only for links its HTTP pass cannot settle,
  # so --version passes with no Chrome and a real run dies partway through.
  local pd; pd="$(puppeteer_dir)"
  if [[ -z "${pd}" ]]; then
    log_error "puppeteer is missing; linkspector cannot check links that need a browser"
    failures=$((failures + 1))
  elif ! grep -q PUPPETEER_CACHE_DIR "$NODE_ENV/bin/linkspector" 2>/dev/null; then
    log_error "$NODE_ENV/bin/linkspector is not the wrapper that sets PUPPETEER_CACHE_DIR"
    failures=$((failures + 1))
  elif (cd "${pd}" && PUPPETEER_CACHE_DIR="$PUPPETEER_CACHE" node -e "
    const args = process.getuid && process.getuid() === 0 ? ['--no-sandbox'] : []
    import('puppeteer').then(p => p.launch({headless: 'new', args})).then(b => b.close())
  ") >/dev/null 2>&1; then
    log_info "✓ linkspector browser (Chrome launches from $PUPPETEER_CACHE)"
  else
    log_error "linkspector's Chrome will not launch"
    log_error "  install it with: (cd ${pd} && PUPPETEER_CACHE_DIR=$PUPPETEER_CACHE node install.mjs)"
    failures=$((failures + 1))
  fi

  if [[ ${path_issues} -gt 0 ]]; then
    echo
    log_warn "${path_issues} tool(s) are off your PATH, or resolve to a copy installed"
    log_warn "elsewhere that may lack the dependencies we need. make targets are"
    log_warn "unaffected: the Makefile puts ${INSTALL_DIR} first itself."
  fi

  if [[ ${failures} -gt 0 ]]; then
    log_error "${failures} tool(s) missing or not working"
    return 1
  fi
}

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

  if [[ "${DIRENV_RELOAD_NEEDED}" == true ]]; then
    echo
    log_info "This shell predates the install; run 'direnv reload' to pick the tools up."
  fi
}

main "$@"
