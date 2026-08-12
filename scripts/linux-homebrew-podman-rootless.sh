#!/usr/bin/env bash

# Set up rootless Podman from Homebrew as a DDEV container provider on Linux,
# using the Docker CLI as a front end, and validate the result.
#
# Usage:
#   linux-homebrew-podman-rootless.sh           Install, configure, then validate
#   linux-homebrew-podman-rootless.sh --check   Validate only; change nothing
#   linux-homebrew-podman-rootless.sh --report  Dump full environment for a bug report
#   linux-homebrew-podman-rootless.sh --help    Show this help
#
# `--report` changes nothing either. Use it when --check flags something whose
# cause is not obvious: it dumps the whole environment (distro, podman
# provenance, every containers.conf, cgroup and systemd state, userns sysctls)
# in one paste-able block, so a maintainer does not have to ask twenty
# questions. It reads no secrets.
#
# `--check` changes nothing and is the fastest way to diagnose a rootless
# Podman setup that has stopped working. It exits nonzero if it finds a
# problem. Most of what it looks for are failures we have actually hit, and
# whose symptoms are unhelpful enough that they cost hours to track down.
#
# `--check` is not Homebrew-specific: it works against a distribution Podman
# too, and a self-consistent distro stack (podman 5 with netavark 1) passes.
# Only the install path assumes Homebrew.
#
# Environment variables:
#   PODMAN_SKIP_SMOKE_TEST       "true" to skip the checks that run a real
#                                container: the smoke test, the image-load
#                                probe and the healthcheck probe. Those are the
#                                slow ones, and also the only ones that test
#                                whether the setup works rather than whether it
#                                looks right, so skip them only in a hurry.

set -o errexit
set -o pipefail
set -o nounset

PODMAN_SKIP_SMOKE_TEST="${PODMAN_SKIP_SMOKE_TEST:-false}"

BREW_PREFIX="${BREW_PREFIX:-/home/linuxbrew/.linuxbrew}"
SMOKE_TEST_IMAGE="${SMOKE_TEST_IMAGE:-ddev/ddev-utilities:latest}"
DOCKER_CONTEXT_NAME="podman-rootless"

problems=0
warnings=0

# Filled in by do_report so report_summary can mention them without re-running
# the (slow) check.
DOCKERCHECK_STATUS="not run"
DOCKERCHECK_BUILD="not run"
IMAGELOAD_STATUS="not run"
HEALTHCHECK_STATUS="not run"

if [ -t 1 ]; then
  red=$'\033[31m'; yellow=$'\033[33m'; green=$'\033[32m'; reset=$'\033[0m'
else
  red=""; yellow=""; green=""; reset=""
fi

ok()      { printf '%s  ok  %s %s\n' "${green}" "${reset}" "$*"; }
warn()    { printf '%s warn %s %s\n' "${yellow}" "${reset}" "$*"; warnings=$((warnings + 1)); }
fail()    { printf '%s FAIL %s %s\n' "${red}" "${reset}" "$*"; problems=$((problems + 1)); }
hint()    { printf '        %s\n' "$*"; }
heading() { printf '\n== %s ==\n' "$*"; }
# do_install only: say what a command is about to do, right before running it,
# so a sudo password prompt never appears out of nowhere.
step()    { printf '  -> %s\n' "$*"; }

# Print the header comment block, so help never drifts from the file.
usage() {
  awk 'NR < 3 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

# Every containers.conf podman reads, in precedence order.
containers_conf_files() {
  local f
  for f in /usr/share/containers/containers.conf \
           /etc/containers/containers.conf \
           /etc/containers/containers.conf.d/*.conf \
           "${HOME}/.config/containers/containers.conf" \
           "${HOME}/.config/containers/containers.conf.d"/*.conf; do
    [ -f "${f}" ] && printf '%s\n' "${f}"
  done
  return 0
}

# The policy.json podman will actually apply. A user-level file overrides the
# system one entirely rather than merging with it.
active_policy_file() {
  local f
  for f in "${HOME}/.config/containers/policy.json" /etc/containers/policy.json \
           /usr/share/containers/policy.json; do
    [ -f "${f}" ] && { printf '%s\n' "${f}"; return 0; }
  done
  return 1
}

# Name the archive transports a policy will reject, or print nothing.
#
# buildx --load hands the built image over as a docker-archive. A policy whose
# default is "reject" only accepts what it lists, so if it lists neither
# docker-archive nor oci-archive, that handover is refused -- while registry
# pulls keep working, because those go through the "docker" transport. The
# result is a machine that pulls, runs and builds, then fails on the last step
# of every build.
policy_archive_gap() {
  local f
  f="$(active_policy_file)" || return 0
  command -v python3 >/dev/null 2>&1 || return 0
  python3 - "${f}" <<'PYEOF'
import json, sys
try:
    with open(sys.argv[1]) as fh:
        policy = json.load(fh)
except Exception:
    sys.exit(0)
if any(r.get("type") == "insecureAcceptAnything" for r in policy.get("default", [])):
    sys.exit(0)  # a permissive default already covers every transport
transports = policy.get("transports", {})
missing = [t for t in ("docker-archive", "oci-archive") if t not in transports]
if missing:
    print(" and ".join(missing))
PYEOF
}

# Resolve the netavark/aardvark-dns that ship alongside the podman binary.
# Podman and its network helpers are versioned together: podman 6.x speaks the
# netavark 2.x network-options schema, podman 5.x speaks netavark 1.x. Feeding
# podman 6.x an older distro netavark fails every container start with the
# thoroughly unhelpful:
#   netavark (exit code 1): failed to load network options:
#   IO error: invalid type: sequence, expected a map
# which the Docker CLI in turn reports only as:
#   unable to upgrade to tcp, received 500
bundled_helper_path() {
  local helper="$1" podman_bin
  podman_bin="$(command -v podman 2>/dev/null)" || return 1
  podman_bin="$(readlink -f "${podman_bin}")"
  printf '%s/libexec/podman/%s' "$(dirname "$(dirname "${podman_bin}")")" "${helper}"
}

# The Go build tags podman was compiled with, read back out of the binary.
#
# Worth knowing because one of them, `systemd`, silently decides whether
# container healthchecks work at all (see check_healthcheck). `podman version`
# does not report tags, but the Go toolchain embeds them, so ask it.
#
# Returns nonzero when the answer is unknown rather than guessing: no Go
# toolchain to ask with, a binary whose build info was stripped (Homebrew's is),
# or a build that passed no -tags at all.
podman_build_tags() {
  local podman_bin
  command -v go >/dev/null 2>&1 || return 1
  podman_bin="$(command -v podman 2>/dev/null)" || return 1
  go version -m "$(readlink -f "${podman_bin}")" 2>/dev/null |
    awk '$1 == "build" && $2 ~ /^-tags=/ {
           sub(/^-tags=/, "", $2); print $2; found = 1
         }
         END { exit !found }'
}

# True when podman would pick the systemd cgroup manager by itself: unified
# cgroup v2, a running systemd, and a user D-Bus to reach it through.
#
# Worth gating on, because asking for systemd where any of the three is missing
# (cgroup v1 host, WSL2 started without systemd) is worse than the cgroupfs
# fallback it replaces: podman refuses to create containers at all.
systemd_cgroups_available() {
  [ -f /sys/fs/cgroup/cgroup.controllers ] &&
    [ -d /run/systemd/system ] &&
    [ -S "/run/user/$(id -u)/bus" ]
}

# ---------------------------------------------------------------------------
# Install / configure
# ---------------------------------------------------------------------------

do_install() {
  heading "Installing rootless Podman (Homebrew)"

  # Stop any rootful Docker so it cannot compete for the socket or for ports.
  step "Stopping any rootful Docker so it can't compete for the socket or ports (sudo)"
  sudo systemctl disable --now docker.service docker.socket 2>/dev/null || true
  sudo rm -f /var/run/docker.sock

  # Remove the distro container stack. netavark and aardvark-dns MUST go too:
  # apt leaves them behind when podman is removed, and a stale
  # /usr/lib/podman/netavark is exactly what breaks a newer Homebrew podman.
  #
  # Deliberately not `apt-get purge podman --auto-remove`: that cascades to
  # every package podman pulled in, including uidmap and fuse-overlayfs,
  # which this rootless setup still needs from the distro.
  step "Removing the distro's podman/crun/netavark/aardvark-dns, if present (sudo)"
  sudo apt-get remove -y podman crun netavark aardvark-dns 2>/dev/null || true

  # newuidmap/newgidmap, which Podman needs to set up the rootless user
  # namespace. Not a Homebrew package: it installs setuid helpers, which have
  # to come from the distro. Check first so a machine that already has them
  # (Fedora ships them by default) doesn't get an unnecessary sudo prompt.
  if ! command -v newuidmap >/dev/null 2>&1 || ! command -v newgidmap >/dev/null 2>&1; then
    step "Installing uidmap for newuidmap/newgidmap (sudo)"
    sudo apt-get install -y uidmap
  fi

  # uidmap is normally pulled in as an automatic dependency of the distro
  # podman package removed above, so apt still marks it "auto" even though
  # this setup now needs it directly. Left alone, a later
  # `apt-get autoremove` -- run for unrelated cleanup -- silently removes it
  # and breaks rootless Podman. Mark it manual whether it was just installed
  # above or was already present.
  step "Protecting uidmap from a future 'apt-get autoremove' (sudo)"
  sudo apt-mark manual uidmap 2>/dev/null || true

  brew install podman >/dev/null
  hash -r

  # Everything below configures the Homebrew podman, so say so loudly when a
  # different one runs instead. GitHub Actions' runner image unpacks a static
  # podman 5.x into /usr/local/bin, and whichever directory comes first in PATH
  # wins -- silently, because that stack is self-consistent and passes --check.
  case "$(command -v podman 2>/dev/null)" in
    "${BREW_PREFIX}"/*) ;;
    *) warn "podman resolves to $(command -v podman), not ${BREW_PREFIX}/bin/podman" ;;
  esac

  # Homebrew's Linux bottles use their own dynamic linker, which does not
  # search the host's /usr/lib/<arch>-linux-gnu paths. netavark and
  # aardvark-dns are Rust binaries that need libgcc_s.so.1 at runtime, and
  # nothing in podman's Homebrew dependency list provides it, so without `gcc`
  # every container start fails with:
  #   error while loading shared libraries: libgcc_s.so.1: cannot open shared object file
  local netavark_bin
  netavark_bin="$(bundled_helper_path netavark || true)"
  if [ -n "${netavark_bin}" ] && [ -x "${netavark_bin}" ] && ! "${netavark_bin}" --version >/dev/null 2>&1; then
    brew install gcc >/dev/null
    hash -r
  fi

  # WSL2 mounts its root filesystem privately: its own boot process mounts it
  # before systemd starts and never marks it shared, unlike a natively booted
  # Linux. Podman then warns on every container start, and nested mounts can
  # go missing. This only lasts for the current WSL2 instance; see the "WSL2:
  # the root filesystem isn't a shared mount" doc warning for how to persist
  # it via /etc/wsl.conf.
  if grep -qi microsoft /proc/version 2>/dev/null; then
    step "Making the WSL2 root filesystem a shared mount (sudo)"
    sudo mount --make-rshared /
  fi

  # Allow binding ports below 1024 so the DDEV router can use 80/443.
  step "Allowing unprivileged processes to bind ports 80/443 (sudo)"
  sudo mkdir -p /etc/sysctl.d
  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee /etc/sysctl.d/60-rootless.conf >/dev/null
  sudo sysctl -p /etc/sysctl.d/60-rootless.conf

  # Without a lingering session, systemd --user has no D-Bus session
  # bus to manage cgroups through, so podman silently falls back to
  # --cgroup-manager=cgroupfs. buildx then pins its buildkit container to the
  # "/docker/buildx" cgroup parent whenever podman reports "cgroupfs" as its
  # driver, which a rootless user can't create outside its own delegated
  # subtree, and every docker-compose build fails with:
  #   crun: create `/sys/fs/cgroup/docker`: Permission denied: OCI permission denied
  # See https://github.com/containers/podman/issues/5443 and
  # https://github.com/containers/podman/pull/29303.
  step "Enabling systemd lingering so podman keeps a session bus (sudo)"
  sudo loginctl enable-linger "$(whoami)"
  for _ in $(seq 1 10); do
    [ -S "/run/user/$(id -u)/bus" ] && break
    sleep 1
  done

  mkdir -p ~/.config/systemd/user
  # If either path is already a symlink -- e.g. from following the
  # "Unit podman.socket could not be found" troubleshooting entry, which
  # points these at Homebrew's own (read-only) shipped units -- `cat >`
  # follows it and tries to overwrite that read-only Cellar file, failing
  # with a permission denied that has nothing to do with file ownership here.
  # Remove whatever is there first so this is idempotent regardless.
  rm -f ~/.config/systemd/user/podman.socket ~/.config/systemd/user/podman.service
  cat > ~/.config/systemd/user/podman.socket <<'EOF'
[Unit]
Description=Podman API Socket
Documentation=man:podman-system-service(1)

[Socket]
ListenStream=%t/podman/podman.sock
SocketMode=0660

[Install]
WantedBy=sockets.target
EOF

  cat > ~/.config/systemd/user/podman.service <<EOF
[Unit]
Description=Podman API Service
Requires=podman.socket
After=podman.socket
Documentation=man:podman-system-service(1)
StartLimitIntervalSec=0

[Service]
Delegate=true
Type=exec
KillMode=process
Environment=LOGGING="--log-level=info"
# systemd --user services start with a minimal PATH that excludes Homebrew's
# bin, even though it's on PATH in an interactive shell. Without this, podman
# execs conmon/crun by searching PATH and a hardcoded list of distro paths,
# neither of which includes Homebrew's, so every container create fails with:
#   could not find a working conmon binary
Environment=PATH=${BREW_PREFIX}/bin:${BREW_PREFIX}/sbin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ExecStart=${BREW_PREFIX}/bin/podman \$LOGGING system service

[Install]
WantedBy=default.target
EOF

  # Fix for Podman 6: "registries.conf must be in v2 format but is in v1"
  step "Writing /etc/containers/registries.conf in the v2 format Podman 6 needs (sudo)"
  sudo mkdir -p /etc/containers
  sudo tee /etc/containers/registries.conf >/dev/null <<'EOF'
unqualified-search-registries = ["docker.io"]
EOF

  systemctl --user daemon-reload

  # Force k8s-file logging instead of the default journald: Homebrew's conmon
  # bottle is built without journald support, so when podman picks journald
  # (its default whenever the systemd journal is readable/writable) conmon
  # fails with "Include journald in compilation path to log to systemd
  # journal" (containers/conmon#348). Also force the file events logger for
  # the same reason.
  mkdir -p ~/.config/containers/containers.conf.d
  {
    echo '[containers]'
    echo 'log_driver = "k8s-file"'
    echo
    echo '[engine]'
    echo 'events_logger = "file"'
    # GitHub Actions' Ubuntu runner image installs podman from the
    # mgoltzsche/podman-static bundle and unpacks the bundle's /etc along with
    # it, so /etc/containers/containers.conf now carries
    # cgroup_manager = "cgroupfs". That outranks podman's own detection, and
    # only a user drop-in outranks /etc, so say systemd explicitly.
    if systemd_cgroups_available; then
      echo 'cgroup_manager = "systemd"'
    fi
    # Same bundle, same problem: its containers.conf.d drop-in pins crun to the
    # statically built /usr/local/bin/crun that ships with podman 5.x. Point
    # back at the crun this podman was built against.
    if [ -x "${BREW_PREFIX}/bin/crun" ]; then
      echo
      echo '[engine.runtimes]'
      printf 'crun = ["%s/bin/crun"]\n' "${BREW_PREFIX}"
    fi
    echo
    echo '[network]'
    # Homebrew's netavark has dropped the "iptables" backend (upstream removed
    # it), but podman's own default still requests "iptables" in some version
    # combinations, causing an intermittent "Must provide a valid firewall
    # backend, got iptables" failure. Force nftables explicitly instead of
    # relying on that mismatched default.
    echo 'firewall_driver = "nftables"'
  } > ~/.config/containers/containers.conf.d/ddev-podman.conf

  # Deliberately do NOT set engine.helper_binaries_dir here. Setting it
  # *replaces* podman's search path rather than extending it, so an entry list
  # that omits podman's own libexec directory makes podman fall back to a
  # distro netavark of the wrong major version (or find no netavark at all).
  # See bundled_helper_path() above.

  mkdir -p ~/.config/containers
  # Native rootless overlay (kernel 5.13+) keeps "Native Overlay Diff" on,
  # which fuse-overlayfs turns off.
  # https://github.com/containers/podman/blob/main/docs/tutorials/performance.md#choosing-a-storage-driver
  cat > ~/.config/containers/storage.conf <<'EOF'
[storage]
driver = "overlay"
EOF

  # Homebrew ships a permissive default policy.json under its own prefix, but
  # nothing links it into any path podman actually searches (~/.config/containers,
  # /etc/containers, /usr/share/containers), so a pull fails outright with
  # "no policy.json file found" until one exists.
  if ! active_policy_file >/dev/null && [ -f "${BREW_PREFIX}/etc/containers/policy.json" ]; then
    cp "${BREW_PREFIX}/etc/containers/policy.json" ~/.config/containers/policy.json
  fi

  systemctl --user enable --now podman.socket

  # Try several times, it can return "failed to reexec: Permission denied"
  podman info --format '{{.Host.RemoteSocket.Path}}' >/dev/null 2>&1 ||
    podman info --format '{{.Host.RemoteSocket.Path}}' >/dev/null 2>&1 || true

  local sock
  sock="$(podman info --format '{{.Host.RemoteSocket.Path}}')"
  docker context rm -f "${DOCKER_CONTEXT_NAME}" >/dev/null 2>&1 || true
  docker context create "${DOCKER_CONTEXT_NAME}" \
    --description "Podman (rootless)" \
    --docker host="unix://${sock}" >/dev/null
  docker context use "${DOCKER_CONTEXT_NAME}" >/dev/null
}

# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

check_podman_present() {
  heading "Podman"
  if ! command -v podman >/dev/null 2>&1; then
    fail "podman is not on PATH"
    hint "Install it with: brew install podman"
    return 1
  fi
  local ver major
  ver="$(podman version --format '{{.Client.Version}}' 2>/dev/null || echo unknown)"
  major="${ver%%.*}"
  if [ "${major}" = "unknown" ] || [ -z "${major}" ]; then
    warn "could not determine podman version"
  elif [ "${major}" -lt 5 ] 2>/dev/null; then
    warn "podman ${ver} is older than 5.0; DDEV works best with 5.0+"
  else
    ok "podman ${ver} at $(command -v podman)"
  fi
  return 0
}

check_socket() {
  heading "Podman API socket"
  if ! systemctl --user is-active podman.socket >/dev/null 2>&1; then
    fail "podman.socket (user) is not active"
    # A `podman` installed by hand via `brew install podman` (rather than this
    # script) leaves its systemd user units under $(brew --prefix)/lib/systemd/user/,
    # which is not one of systemctl --user's search paths, so the unit is
    # simply unknown rather than failed.
    if systemctl --user status podman.socket 2>&1 | grep -q 'could not be found'; then
      hint "systemctl --user does not know a podman.socket unit at all."
      hint "On a Homebrew install, its units live under \$(brew --prefix)/lib/systemd/user/,"
      hint "which is not one of systemctl --user's search paths. Link them in:"
      hint "  mkdir -p ~/.config/systemd/user"
      hint "  ln -s \"\$(brew --prefix)\"/lib/systemd/user/podman.{socket,service} ~/.config/systemd/user/"
      hint "  systemctl --user daemon-reload"
      hint "  systemctl --user enable --now podman.socket"
    # A run of failed activations (a broken netavark makes podman exit 125 on
    # every connection) trips the systemd start limit and leaves both units in
    # "failed". Fixing the underlying config does NOT clear that by itself:
    # without reset-failed, `start` refuses and the stale socket file makes the
    # Docker CLI report "Cannot connect to the Docker daemon".
    elif systemctl --user is-failed podman.socket >/dev/null 2>&1 ||
       systemctl --user is-failed podman.service >/dev/null 2>&1; then
      hint "The unit is in 'failed' state, so it must be reset before it will start:"
      hint "  systemctl --user reset-failed podman.socket podman.service"
      hint "  systemctl --user start podman.socket"
      hint "Check why it failed first: journalctl --user -u podman.service --since '-15min'"
    else
      hint "systemctl --user enable --now podman.socket"
    fi
    return 0
  fi
  ok "podman.socket is active"

  local sock
  sock="$(podman info --format '{{.Host.RemoteSocket.Path}}' 2>/dev/null || true)"
  if [ -z "${sock}" ] || [ ! -S "${sock}" ]; then
    fail "podman reports no usable API socket"
    return 0
  fi
  ok "API socket ${sock}"

  if ! loginctl show-user "$(whoami)" --property=Linger 2>/dev/null | grep -q 'Linger=yes'; then
    warn "lingering is not enabled for $(whoami)"
    hint "sudo loginctl enable-linger $(whoami)"
    hint "Without it podman falls back to the cgroupfs cgroup manager and"
    hint "docker-compose builds fail with 'create /sys/fs/cgroup/docker: Permission denied'."
  else
    ok "systemd lingering is enabled"
  fi
}

# The check that would have saved the most time: podman resolving a network
# helper of a different major version than the one it shipped with.
check_helper_versions() {
  heading "Network helpers (netavark / aardvark-dns)"

  # Run the bundled binaries directly, before asking podman about them. On a
  # Homebrew install, they can fail to execute at all -- Homebrew's Linux
  # dynamic linker doesn't search the host's system library paths, and these
  # Rust binaries need libgcc_s.so.1 at runtime, which nothing in podman's
  # Homebrew dependency list provides. podman info's NetworkBackendInfo fields
  # come back empty in that case, which is a far less direct diagnostic than
  # the loader's own error.
  local helper bundled errmsg helper_broken=0
  for helper in netavark aardvark-dns; do
    bundled="$(bundled_helper_path "${helper}" || true)"
    [ -n "${bundled}" ] && [ -x "${bundled}" ] || continue
    if ! errmsg="$("${bundled}" --version 2>&1)"; then
      helper_broken=1
      fail "${bundled} fails to run: $(printf '%s' "${errmsg}" | tail -1)"
      if printf '%s' "${errmsg}" | grep -q 'libgcc_s'; then
        hint "Homebrew's own dynamic linker does not search the host's system"
        hint "library paths, and nothing in podman's Homebrew dependency list"
        hint "provides libgcc_s.so.1, which ${helper} needs at runtime."
        hint "Fix: brew install gcc"
      fi
    fi
  done
  if [ "${helper_broken}" -eq 1 ]; then
    return 0
  fi

  local info resolved_netavark netavark_ver resolved_dns dns_ver
  info="$(podman info \
    --format '{{.Host.NetworkBackendInfo.Path}}|{{.Host.NetworkBackendInfo.Version}}|{{.Host.NetworkBackendInfo.DNS.Path}}|{{.Host.NetworkBackendInfo.DNS.Version}}' \
    2>&1 || true)"

  if printf '%s' "${info}" | grep -q 'could not find'; then
    fail "podman cannot find its network helpers"
    hint "${info}"
    hint "This is usually engine.helper_binaries_dir in containers.conf pointing"
    hint "at directories that do not contain netavark. Remove that setting."
    return 0
  fi

  IFS='|' read -r resolved_netavark netavark_ver resolved_dns dns_ver <<< "${info}"

  local helper
  for helper in netavark aardvark-dns; do
    local bundled resolved ver
    bundled="$(bundled_helper_path "${helper}" || true)"
    if [ "${helper}" = "netavark" ]; then
      resolved="${resolved_netavark}"; ver="${netavark_ver}"
    else
      resolved="${resolved_dns}"; ver="${dns_ver}"
    fi

    if [ -z "${resolved}" ]; then
      fail "podman did not report a path for ${helper}"
      continue
    fi

    if [ -n "${bundled}" ] && [ -x "${bundled}" ] &&
       [ "$(readlink -f "${bundled}")" != "$(readlink -f "${resolved}")" ]; then
      fail "podman is using ${resolved} (${ver})"
      hint "but it ships with ${bundled} ($("${bundled}" --version 2>/dev/null || echo '?'))"
      hint "Mismatched major versions break every container start. Remove any"
      hint "engine.helper_binaries_dir setting from your containers.conf files,"
      hint "and remove the distro package: sudo apt-get purge ${helper}"
      continue
    fi

    ok "${ver} at ${resolved}"
  done

  # Explicit pairing check, for distro installs that have no bundled helpers
  # next to the podman binary. A self-consistent distro stack (podman 5 with
  # netavark 1, podman 4 with netavark 1) is fine; only skew is a problem.
  local podman_major netavark_major
  podman_major="$(podman version --format '{{.Client.Version}}' 2>/dev/null | cut -d. -f1)"
  netavark_major="$(printf '%s' "${netavark_ver}" | awk '{print $2}' | cut -d. -f1)"
  if [ -n "${podman_major}" ] && [ -n "${netavark_major}" ] 2>/dev/null; then
    if [ "${podman_major}" -ge 6 ] && [ "${netavark_major}" -lt 2 ]; then
      fail "podman ${podman_major}.x needs netavark 2.x, found ${netavark_ver}"
      hint "Symptom: 'failed to load network options: invalid type: sequence, expected a map'"
      hint "Symptom via the Docker CLI: 'unable to upgrade to tcp, received 500'"
    elif [ "${podman_major}" -le 5 ] && [ "${netavark_major}" -ge 2 ]; then
      fail "podman ${podman_major}.x needs netavark 1.x, found ${netavark_ver}"
      hint "Either upgrade podman or reinstall your distribution's netavark."
    fi
  fi

  # The loaded gun: a foreign netavark sitting where podman can find it.
  # /usr/lib/podman belongs to the distro package; /usr/local/lib/podman is
  # where GitHub Actions' runner image unpacks the podman-static bundle. Both
  # are on podman's default helper search path.
  local stale
  for stale in /usr/lib/podman/netavark /usr/local/lib/podman/netavark; do
    [ -e "${stale}" ] || continue
    [ "$(readlink -f "${resolved_netavark}")" = "$(readlink -f "${stale}")" ] && continue
    warn "an unused netavark remains at ${stale}"
    hint "$("${stale}" --version 2>/dev/null || echo 'version unknown')"
    hint "Not currently in use, but any helper_binaries_dir change can select it."
  done
}

# The OCI runtime has the same failure mode as the network helpers -- podman and
# crun are released as a pair -- but a different resolution path, so a pin in
# someone else's containers.conf is easy to miss.
check_oci_runtime() {
  heading "OCI runtime"

  local resolved expected podman_bin
  resolved="$(podman info --format '{{.Host.OCIRuntime.Path}}' 2>/dev/null || true)"
  if [ -z "${resolved}" ]; then
    warn "podman did not report an OCI runtime path"
    return 0
  fi

  # Same reasoning as bundled_helper_path: what shipped with this podman sits
  # under the same prefix as the podman binary.
  podman_bin="$(readlink -f "$(command -v podman)")"
  expected="$(dirname "${podman_bin}")/$(basename "${resolved}")"

  if [ -x "${expected}" ] &&
     [ "$(readlink -f "${expected}")" != "$(readlink -f "${resolved}")" ]; then
    warn "podman is using ${resolved}"
    hint "but ${expected} ships with this podman"
    hint "Something pins it, usually [engine.runtimes] in a containers.conf under"
    hint "/etc. Override it in ~/.config/containers/containers.conf.d/ddev-podman.conf."
    return 0
  fi

  ok "$(basename "${resolved}") at ${resolved}"
}

check_helper_binaries_dir() {
  heading "containers.conf overrides"
  local found=0 f
  for f in /etc/containers/containers.conf /etc/containers/containers.conf.d/*.conf \
           "${HOME}/.config/containers/containers.conf" \
           "${HOME}/.config/containers/containers.conf.d"/*.conf; do
    [ -f "${f}" ] || continue
    if grep -Eq '^[[:space:]]*helper_binaries_dir[[:space:]]*=' "${f}"; then
      found=1
      fail "helper_binaries_dir is set in ${f}"
      hint "$(grep -E '^[[:space:]]*helper_binaries_dir[[:space:]]*=' "${f}")"
      hint "This REPLACES podman's helper search path rather than extending it."
      hint "Unless the list contains podman's own libexec directory, podman will"
      hint "use a wrong-version helper or none at all. Remove the setting."
    fi
  done
  [ "${found}" -eq 0 ] && ok "no helper_binaries_dir override"
  return 0
}

# The Docker CLI and an interactive shell both find conmon/crun via PATH, but
# `systemd --user` services start with a minimal PATH that excludes wherever a
# non-distro podman's helpers live (Homebrew's bin, an unpackaged install,
# etc.). podman.service then execs them by searching PATH plus a fixed list of
# distro paths, finds neither, and every container create fails with:
#   could not find a working conmon binary
# `podman info` still reports the right conmon path, because that comes from
# the CLI's own PATH, not the service's -- which is exactly what makes this
# confusing to diagnose from the client side.
check_service_path() {
  heading "podman.service PATH (conmon lookup)"
  local conmon_path conmon_dir unit_path svc_path
  conmon_path="$(command -v conmon 2>/dev/null || true)"
  if [ -z "${conmon_path}" ]; then
    warn "conmon not found on PATH; skipped"
    return 0
  fi
  conmon_dir="$(dirname "${conmon_path}")"

  # podman's own fixed fallback list, mirrored from libpod's default config.
  case ":/usr/libexec/podman:/usr/local/libexec/podman:/usr/local/lib/podman:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:/run/current-system/sw/bin:" in
    *":${conmon_dir}:"*)
      ok "conmon (${conmon_dir}) is in podman's built-in search list"
      return 0
      ;;
  esac

  unit_path="$(systemctl --user show podman.service --property=Environment 2>/dev/null)"
  svc_path="$(systemctl --user show-environment 2>/dev/null | grep '^PATH=' || true)"
  if printf '%s\n%s' "${unit_path}" "${svc_path}" | grep -q "${conmon_dir}"; then
    ok "conmon (${conmon_dir}) is reachable via podman.service's PATH"
  else
    fail "conmon lives at ${conmon_dir}, which is in neither podman's built-in"
    hint "search list nor podman.service's PATH (or its own Environment=)."
    hint "Add it to the service unit: systemctl --user edit podman.service, then"
    hint "under [Service]: Environment=PATH=${conmon_dir}:/usr/bin:/bin"
    hint "systemctl --user daemon-reload && systemctl --user restart podman.socket"
  fi
}

# Without one, a pull fails outright rather than falling back to any default:
#   config file not found: no policy.json file found; searched paths: [...]
# A Homebrew install ships a permissive default under its own prefix, but
# nothing links it into any of the paths podman actually searches.
check_policy() {
  heading "Image signature policy"
  local f
  if f="$(active_policy_file)"; then
    ok "policy.json at ${f}"
    return 0
  fi
  fail "no policy.json found in any path podman searches"
  hint "~/.config/containers/policy.json, /etc/containers/policy.json, /usr/share/containers/policy.json"
  if [ -f "${BREW_PREFIX}/etc/containers/policy.json" ]; then
    hint "Homebrew's podman ships a default one; use it:"
    hint "  mkdir -p ~/.config/containers"
    hint "  cp ${BREW_PREFIX}/etc/containers/policy.json ~/.config/containers/policy.json"
  else
    hint "A permissive default accepts every transport:"
    hint '  {"default": [{"type": "insecureAcceptAnything"}]}'
  fi
}

check_cgroups() {
  heading "cgroups"
  local mgr
  mgr="$(podman info --format '{{.Host.CgroupManager}}' 2>/dev/null || echo unknown)"
  if [ "${mgr}" = "systemd" ]; then
    ok "cgroup manager is systemd"
    return 0
  fi

  fail "cgroup manager is '${mgr}', expected 'systemd'"
  hint "This can break docker-compose builds with:"
  hint "  crun: create \`/sys/fs/cgroup/docker\`: Permission denied"
  hint "It does not always: forcing cgroupfs on podman 6.0.2 still built fine."
  hint "Confirm whether that is really happening to you before chasing this:"
  hint "  ddev debug dockercheck    # runs a trivial buildx build"
  hint "If that build succeeds, cgroupfs is not what is blocking you."

  # Work out *why* rather than guessing. Lingering is only one of several
  # causes, and blaming it when it is already enabled sends people in circles.

  # 1. Explicitly configured. This wins over everything, so report it first.
  local f line found=0
  while read -r f; do
    line="$(grep -nE '^[[:space:]]*cgroup_manager[[:space:]]*=' "${f}" 2>/dev/null || true)"
    if [ -n "${line}" ]; then
      found=1
      hint "set explicitly in ${f}: ${line}"
    fi
  done < <(containers_conf_files)
  if [ "${found}" -eq 1 ]; then
    hint "Remove that setting, or -- when the file belongs to the system, as on"
    hint "a GitHub Actions runner -- override it with cgroup_manager = \"systemd\""
    hint "under [engine] in ~/.config/containers/containers.conf.d/ddev-podman.conf,"
    hint "which outranks everything in /etc. Running this script without --check"
    hint "writes that for you."
    return 0
  fi

  # 2. Find out whether podman is *able* to use the systemd manager. Asking
  #    `podman --cgroup-manager=systemd info` proves nothing: info just echoes
  #    the flag back, and only rejects an invalid value. Actually running a
  #    container under that manager is the only honest test.
  if [ "${PODMAN_SKIP_SMOKE_TEST}" != "true" ]; then
    local probe
    if probe="$(podman --cgroup-manager=systemd run --rm "${SMOKE_TEST_IMAGE}" true 2>&1)"; then
      hint "podman CAN run under the systemd manager here, so nothing is stopping"
      hint "it -- podman's own default just selected cgroupfs, which normally"
      hint "means it saw no usable systemd user session when it started."
      hint "Compare the CLI with the service, which is what builds actually use:"
      hint "  podman info --format '{{.Host.CgroupManager}}'"
      hint "  docker info --format '{{.CgroupDriver}}'"
      hint "If they differ, restart the service: systemctl --user restart podman.socket"
    else
      hint "podman fails to run a container under the systemd manager:"
      hint "  $(printf '%s' "${probe}" | tail -1)"
    fi
  fi

  # 3. Environment prerequisites, mentioned only when actually wrong.
  if [ ! -S "/run/user/$(id -u)/bus" ]; then
    hint "No session bus at /run/user/$(id -u)/bus; systemd --user is not usable."
  fi
  if ! loginctl show-user "$(whoami)" --property=Linger 2>/dev/null | grep -q 'Linger=yes'; then
    hint "Lingering is off: sudo loginctl enable-linger $(whoami)"
  fi
}

check_ports() {
  heading "Privileged ports"
  local start
  start="$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null || echo unknown)"
  # The sysctl is the *lowest* port an unprivileged process may bind, so any
  # value <= 80 leaves both 80 and 443 usable. Only a higher value is a problem.
  if ! [ "${start}" -le 80 ] 2>/dev/null; then
    warn "net.ipv4.ip_unprivileged_port_start is ${start}, so ports 80/443 are unavailable"
    hint "Either allow low ports:"
    hint "  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee /etc/sysctl.d/60-rootless.conf"
    hint "  sudo sysctl -p /etc/sysctl.d/60-rootless.conf"
    hint "or configure DDEV for unprivileged ports:"
    hint "  ddev config global --router-http-port=8080 --router-https-port=8443"
  else
    ok "unprivileged processes may bind ports 80/443"
  fi
}

# WSL2-specific: its own boot process mounts the root filesystem before
# systemd starts and never marks it shared, unlike a natively booted Linux
# where it's shared from boot. podman then warns on every container start
# ("/" is not a shared mount...) and nested mounts can go missing. Not known
# to break typical DDEV usage, but cheap to flag and fix.
check_mount_propagation() {
  heading "Root mount propagation"
  local prop
  prop="$(findmnt -no PROPAGATION / 2>/dev/null || true)"
  if [ -z "${prop}" ]; then
    warn "could not determine mount propagation of /"
    return 0
  fi
  if printf '%s' "${prop}" | grep -q shared; then
    ok "/ is a shared mount"
    return 0
  fi
  warn "/ is a '${prop}' mount, not shared"
  hint "podman warns: \"/\" is not a shared mount, this could cause issues or"
  hint "missing mounts with rootless containers"
  if grep -qi microsoft /proc/version 2>/dev/null; then
    hint "Expected on WSL2. Fix for the current instance: sudo mount --make-rshared /"
    hint "That reverts on every 'wsl --shutdown'. To persist it, add to /etc/wsl.conf:"
    hint "  [boot]"
    hint "  command = mount --make-rshared /"
  else
    hint "Fix: sudo mount --make-rshared /"
  fi
}

check_subuid() {
  heading "subuid / subgid"
  local user f
  user="$(id -un)"

  # newuidmap/newgidmap come from the uidmap package on Debian/Ubuntu (Fedora
  # ships them by default via shadow-utils). Without them podman cannot set up
  # the rootless user namespace at all.
  if ! command -v newuidmap >/dev/null 2>&1 || ! command -v newgidmap >/dev/null 2>&1; then
    fail "newuidmap/newgidmap not found"
    hint "sudo apt-get install -y uidmap   # Fedora: shadow-utils (usually already installed)"
  else
    ok "newuidmap/newgidmap present"
    # apt marks a package "auto" once nothing manually-installed depends on
    # it -- typical after removing a distro podman that pulled uidmap in.
    # A later `apt-get autoremove`, run for unrelated cleanup, then removes it
    # and breaks rootless Podman.
    if command -v apt-mark >/dev/null 2>&1 && apt-mark showauto uidmap 2>/dev/null | grep -qx uidmap; then
      warn "uidmap is marked auto-installed"
      hint "a future 'apt-get autoremove' could remove it and break rootless Podman"
      hint "fix: sudo apt-mark manual uidmap"
    fi
  fi

  for f in /etc/subuid /etc/subgid; do
    if ! grep -q "^${user}:" "${f}" 2>/dev/null; then
      fail "no entry for ${user} in ${f}"
      hint "sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 ${user}"
      continue
    fi
    # Flag ranges that overlap another user's, which collides as soon as a
    # second user (or rootful podman) starts mapping the same subordinate IDs.
    # A user may legitimately have several ranges, so compare every range
    # against every other rather than keeping one range per user.
    local overlap
    overlap="$(awk -F: -v me="${user}" '
      { u[NR]=$1; s[NR]=$2+0; l[NR]=$3+0; n=NR }
      END {
        for (i = 1; i <= n; i++) {
          if (u[i] != me) continue
          for (j = 1; j <= n; j++) {
            if (u[j] == me) continue
            if (s[i] < s[j]+l[j] && s[j] < s[i]+l[i])
              printf "%s:%d-%d ", u[j], s[j], s[j]+l[j]-1
          }
        }
      }' "${f}" | tr ' ' '\n' | sort -u | tr '\n' ' ')"
    if [ -n "${overlap}" ]; then
      warn "${user}'s range in ${f} overlaps: ${overlap}"
    else
      ok "${f} has a non-overlapping range for ${user}"
    fi
  done
}

check_docker_cli() {
  heading "Docker CLI front end"
  if ! command -v docker >/dev/null 2>&1; then
    fail "docker CLI is not on PATH"
    hint "Install just the CLI (not the engine): sudo apt-get install docker-ce-cli"
    return 0
  fi

  local ctx endpoint sock
  ctx="$(docker context show 2>/dev/null || echo unknown)"
  endpoint="$(docker context inspect "${ctx}" --format '{{.Endpoints.docker.Host}}' 2>/dev/null || echo '')"
  sock="$(podman info --format '{{.Host.RemoteSocket.Path}}' 2>/dev/null || true)"

  if [ -n "${sock}" ] && [ "${endpoint}" = "unix://${sock}" ]; then
    ok "context '${ctx}' -> ${endpoint}"
  else
    fail "active docker context '${ctx}' points at ${endpoint:-nothing}, not unix://${sock}"
    hint "docker context use ${DOCKER_CONTEXT_NAME}"
  fi

  # A second engine running at the same time contends for ports 80/443.
  if systemctl is-active docker.service >/dev/null 2>&1; then
    warn "rootful docker.service is running alongside podman"
    hint "sudo systemctl disable --now docker.service docker.socket"
  fi
  if systemctl --user is-active docker.service >/dev/null 2>&1; then
    warn "rootless docker.service (user) is running alongside podman"
    hint "systemctl --user disable --now docker.service"
  fi
}

# DDEV builds its project images with `docker buildx`, and the default
# global config (docker_buildx_version: "system") means DDEV will NOT download
# a buildx for you. Installing only docker-ce-cli leaves buildx missing, and
# `ddev start` then fails at the build step rather than at startup.
check_buildx() {
  heading "docker buildx"
  local min="0.17.0" ver path
  if ! command -v docker >/dev/null 2>&1; then
    warn "skipped, no docker CLI"
    return 0
  fi
  ver="$(docker buildx version 2>/dev/null | awk '{print $2}' | sed 's/^v//' | cut -d+ -f1)"
  if [ -z "${ver}" ]; then
    fail "the docker buildx plugin is not installed"
    hint "DDEV requires buildx ${min}+ to build project images."
    hint "sudo apt-get install -y docker-buildx-plugin"
    hint "  (or: sudo dnf install docker-buildx-plugin)"
    return 0
  fi
  if [ "$(printf '%s\n%s\n' "${min}" "${ver}" | sort -V | head -1)" != "${min}" ]; then
    fail "buildx ${ver} is older than the required ${min}"
    hint "sudo apt-get install -y docker-buildx-plugin"
    return 0
  fi
  path="$(docker buildx version 2>/dev/null | awk '{print $1}')"
  ok "buildx ${ver} (${path})"
}

check_storage() {
  heading "Storage driver"
  local driver native
  driver="$(podman info --format '{{.Store.GraphDriverName}}' 2>/dev/null || echo unknown)"
  native="$(podman info --format '{{index .Store.GraphStatus "Native Overlay Diff"}}' 2>/dev/null || echo unknown)"
  if [ "${driver}" != "overlay" ]; then
    warn "storage driver is '${driver}'; 'overlay' performs best"
  elif [ "${native}" = "false" ]; then
    warn "overlay is in use but native overlay diff is off (fuse-overlayfs)"
    hint "On kernel 5.13+ native rootless overlay is supported and faster."
    hint "Drop mount_program from ~/.config/containers/storage.conf, then"
    hint "'podman system reset' (this deletes all images/containers/volumes)."
  else
    ok "overlay with native overlay diff"
  fi
}

check_smoke_test() {
  heading "Smoke test"
  if [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    IMAGELOAD_STATUS="skipped"
    ok "skipped (PODMAN_SKIP_SMOKE_TEST=true)"
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    warn "skipped, no docker CLI"
    return 0
  fi
  local out
  # Exercises image pull, container create, network setup and attach, which is
  # the path a broken netavark breaks.
  if out="$(docker run --rm "${SMOKE_TEST_IMAGE}" sh -c 'echo container-ok' 2>&1)" &&
     printf '%s' "${out}" | grep -q container-ok; then
    ok "started a container and attached to it"
  else
    fail "could not run a container"
    hint "${out}"
    if printf '%s' "${out}" | grep -qi 'unauthorized\|incorrect username or password'; then
      hint "Stale or invalid Docker Hub credentials interfere even with pulling"
      hint "public images. Fix: docker logout"
    else
      hint "The Docker CLI hides the real error. See the server side with:"
      hint "  journalctl --user -u podman.service --since '-5min'"
    fi
  fi
}

# The last thing `ddev start` does when building a project image is hand the
# result back to the engine to load, and that load is a separate operation with
# its own failure mode: containers/image applies the signature policy to it, so
# a policy that does not accept the docker-archive transport rejects it. The
# build succeeds in full and only the final step fails:
#   failed to load image: payload does not match any of the supported image
#   formats: ... Source image rejected: ... is rejected by policy.
#
# Nothing else here reaches that path. `podman run` never loads an archive, and
# `ddev debug dockercheck` builds without --load, so with the docker-container
# driver its result stays in the build cache and is never handed to the engine
# ("WARNING: No output specified with docker-container driver"). A machine can
# therefore pull, run, build and pass dockercheck and still not finish
# `ddev start`.
#
# So probe the real path: buildx with --load, from an empty image. No base
# image to pull, about a second. Fall back to podman save/load when buildx is
# unavailable, which covers the same policy but via the CLI rather than the API.
check_image_load() {
  heading "Image load (the step after a build)"
  if [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    ok "skipped (PODMAN_SKIP_SMOKE_TEST=true)"
    return 0
  fi

  local tag="ddev-load-probe" ctx tar out
  ctx="$(mktemp -d)"
  tar="$(mktemp -u)".tar

  # A signature policy can reject at build, save or load, so treat a rejection
  # anywhere as the same finding rather than skipping the check.
  _probe_failed() {
    local step="$1" msg="$2"
    if printf '%s' "${msg}" | grep -q 'rejected by policy'; then
      IMAGELOAD_STATUS="FAIL (rejected by policy)"
      fail "the image signature policy rejects images (at ${step})"
      hint "This is what makes 'ddev start' fail at the very end, after the"
      hint "build has already succeeded, with:"
      hint "  failed to load image: ... Source image rejected: ... rejected by policy"
      local gap; gap="$(policy_archive_gap)"
      if [ -n "${gap}" ]; then
        # A deliberate allowlist policy: name the gap and keep the rest intact,
        # rather than telling someone to throw their whole policy away.
        hint "$(active_policy_file) rejects by default and allows no ${gap}."
        hint "Add just the transports buildx --load needs, leaving the rest as-is:"
        hint '  "docker-archive": { "": [{ "type": "insecureAcceptAnything" }] },'
        hint '  "oci-archive":    { "": [{ "type": "insecureAcceptAnything" }] }'
      else
        hint "Check the default in whichever of these exists:"
        hint "  ${HOME}/.config/containers/policy.json   (takes precedence)"
        hint "  /etc/containers/policy.json"
        hint "A permissive default accepts every transport:"
        hint '  {"default": [{"type": "insecureAcceptAnything"}]}'
      fi
    else
      IMAGELOAD_STATUS="FAIL (at ${step})"
      fail "a built image could not be loaded into the engine (at ${step})"
      hint "$(printf '%s' "${msg}" | tail -3)"
    fi
  }

  if command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1; then
    if out="$(printf 'FROM scratch\n' | docker buildx build --load -f - -t "${tag}" "${ctx}" 2>&1)"; then
      IMAGELOAD_STATUS="ok"
      ok "buildx can build an image and load it into the engine"
    else
      _probe_failed "buildx --load" "${out}"
    fi
    docker rmi -f "${tag}" >/dev/null 2>&1 || true
  elif ! out="$(printf 'FROM scratch\n' | podman build -q -t "${tag}" -f - "${ctx}" 2>&1)"; then
    _probe_failed "build" "${out}"
  elif ! out="$(podman save -o "${tar}" "${tag}" 2>&1)"; then
    _probe_failed "save" "${out}"
  elif ! out="$(podman load -i "${tar}" 2>&1)"; then
    _probe_failed "load" "${out}"
  else
    IMAGELOAD_STATUS="ok"
    ok "a built image can be saved and loaded back into podman"
    podman rmi -f "${tag}" >/dev/null 2>&1 || true
  fi

  rm -rf "${ctx}" "${tar}"
  unset -f _probe_failed
}

# `ddev start` does not poll a container's port to decide it is up; it waits for
# the container's own HEALTHCHECK to report healthy. So a healthcheck that never
# runs stalls every start until the timeout, and does it without producing an
# error to work from:
#   ddev-<project>-web failed to become ready ... timed out without becoming healthy
# with a health log that is empty rather than failing:
#   {"Status":"starting","FailingStreak":0,"Log":null}
#
# Podman does not run healthchecks in-process. For each container it registers a
# transient systemd *user* timer, `<container-id>-<hash>.timer`, which fires
# `podman healthcheck run <id>` on an interval. Two things therefore have to hold
# beyond the healthcheck command itself working:
#
#   1. podman must have been compiled with its `systemd` build tag. Without it,
#      libpod compiles healthcheck_nosystemd_linux.go, whose createTimer() and
#      startTimer() `return nil` and do nothing at all. Podman still records the
#      HealthConfig, so the container reports "starting" forever and never logs
#      an attempt. Nothing warns at runtime.
#   2. the systemd user session must accept transient units (lingering on, user
#      bus present, `systemd-run --user` working).
#
# The build-tag case is easy to hit when building podman from source, because
# `make BUILDTAGS="..."` *replaces* the tags podman's Makefile computes rather
# than adding to them, and the only warning it prints frames the consequence as
# losing "journald support".
#
# Running the healthcheck script by hand proves nothing about any of this: it
# passes on a machine where healthchecks never run. Only watching a real
# container reach `healthy` tests it, so that is what this does.
check_healthcheck() {
  heading "Container healthchecks (what 'ddev start' waits for)"
  if [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    HEALTHCHECK_STATUS="skipped"
    ok "skipped (PODMAN_SKIP_SMOKE_TEST=true)"
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    HEALTHCHECK_STATUS="skipped (no docker CLI)"
    warn "skipped, no docker CLI"
    return 0
  fi

  local name="ddev-healthcheck-probe" cid status="" log tags waited=0
  docker rm -f "${name}" >/dev/null 2>&1 || true
  # Go through the Docker CLI, because that is the API DDEV uses to create
  # containers; a healthcheck registered over the compat API is the case that
  # matters here.
  if ! cid="$(docker run -d --name "${name}" \
      --health-cmd 'true' --health-interval 2s --health-retries 1 \
      "${SMOKE_TEST_IMAGE}" sleep 60 2>&1)"; then
    HEALTHCHECK_STATUS="FAIL (probe container would not start)"
    fail "could not start a container to test healthchecks with"
    hint "$(printf '%s' "${cid}" | tail -2)"
    return 0
  fi

  while [ "${waited}" -lt 20 ]; do
    status="$(docker inspect --format '{{.State.Health.Status}}' "${name}" 2>/dev/null || true)"
    [ "${status}" = "healthy" ] && break
    sleep 1
    waited=$((waited + 1))
  done

  if [ "${status}" = "healthy" ]; then
    HEALTHCHECK_STATUS="ok"
    ok "a container healthcheck ran and reported healthy (${waited}s)"
  else
    HEALTHCHECK_STATUS="FAIL (stuck at '${status:-unknown}')"
    fail "healthchecks do not run: after ${waited}s the container is still '${status:-unknown}'"
    hint "Every 'ddev start' will time out waiting for web and db to become"
    hint "healthy, however well the containers themselves are running."

    # Whether podman scheduled a timer at all separates the two failure shapes,
    # so establish that before saying what an empty log means.
    local timer=no
    systemctl --user list-units --all --no-pager "${cid:0:32}*" 2>/dev/null |
      grep -q '\.timer' && timer=yes

    log="$(docker inspect --format '{{json .State.Health.Log}}' "${name}" 2>/dev/null || true)"
    if [ "${log}" = "null" ] || [ "${log}" = "[]" ] || [ -z "${log}" ]; then
      if [ "${timer}" = "yes" ]; then
        hint "The health log is empty even though a timer exists, so a run was"
        hint "scheduled and has not returned: it is hanging, not failing."
      else
        hint "The health log is empty, so nothing ever executed the healthcheck;"
        hint "the healthcheck command itself is not the problem."
      fi
    else
      hint "The healthcheck did run and failed. Its output:"
      hint "${log}"
    fi

    # Cause order. A missing build tag makes this unconditional and silent, so
    # rule it in or out before looking at the systemd session.
    if [ "${timer}" = "no" ]; then
      hint "No healthcheck timer unit exists for the probe container, so podman"
      hint "never even tried to schedule one."
      if tags="$(podman_build_tags)"; then
        if printf '%s' "${tags}" | tr ',' '\n' | grep -qx systemd; then
          hint "podman does have its 'systemd' build tag, so look at the session:"
        else
          HEALTHCHECK_STATUS="FAIL (podman built without its systemd build tag)"
          hint "podman was compiled WITHOUT its 'systemd' build tag:"
          hint "  -tags=${tags}"
          hint "That is the cause: podman's healthcheck timer functions are"
          hint "compiled as no-ops, so no container can ever become healthy."
          hint "Rebuild podman and let its Makefile choose the tags:"
          hint "  make PREFIX=/usr/local                     # detects systemd"
          hint "  make PREFIX=/usr/local EXTRA_BUILDTAGS=... # to add your own"
          hint "'make BUILDTAGS=...' REPLACES the computed defaults and drops"
          hint "systemd along with them. libsystemd-dev (Debian/Ubuntu) or"
          hint "systemd-devel (RPM) must be installed for it to be detected."
        fi
      else
        hint "Could not read podman's build tags. If podman was built from"
        hint "source, check that 'systemd' is among them: a build without it"
        hint "makes healthchecks silent no-ops. Check with:"
        hint "  go version -m \$(command -v podman) | grep -- -tags="
      fi
      if ! command -v systemd-run >/dev/null 2>&1; then
        hint "systemd-run is not on PATH; podman needs it to create the timer."
      elif ! systemd-run --user --quiet --collect --unit="ddev-hc-probe-$$" \
             /bin/true >/dev/null 2>&1; then
        hint "This session cannot create transient systemd user units at all"
        hint "('systemd-run --user' failed), which podman needs for the timer."
      fi
    else
      hint "A timer unit exists, so podman's scheduling side works and the"
      hint "problem is in the run it triggers. Inspect both with:"
      hint "  systemctl --user list-timers --all | grep ${cid:0:12}"
      hint "  journalctl --user --since '-5min' | grep ${cid:0:12}"
    fi
  fi

  docker rm -f "${name}" >/dev/null 2>&1 || true
}

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

# Dump everything needed to diagnose someone else's machine in one paste.
# Deliberately gathers facts rather than judging them: --check says what looks
# wrong, --report says what is actually there. Reads nothing secret (auth.json
# and ssh keys are never touched).
do_report() {
  # A dump must never abort partway: a command that fails (a missing path, a
  # tool that isn't installed) is itself a useful datum, not a reason to stop.
  set +o errexit
  set +o pipefail

  local u; u="$(id -un)"
  sec() { printf '\n----- %s -----\n' "$*"; }
  run() {
    printf '$ %s\n' "$*"
    eval "$*" 2>&1 | sed 's/^/  /'
    local rc=${PIPESTATUS[0]}
    [ "${rc}" -ne 0 ] && printf '  [exit %s]\n' "${rc}"
    return 0
  }

  printf '===== rootless podman report for DDEV =====\n'

  sec "host"
  run "grep -E '^(NAME|VERSION|VERSION_ID|ID|ID_LIKE|VARIANT)=' /etc/os-release"
  run "uname -srmo"
  run "systemctl --version | head -1"

  sec "podman binary and provenance"
  run "command -v podman"
  run "readlink -f \$(command -v podman)"
  run "podman version --format '{{.Client.Version}} BuildOrigin={{.Client.BuildOrigin}} go={{.Client.GoVersion}}'"
  # How it was installed decides who owns the config defaults under /usr/share.
  run "dpkg -S \$(readlink -f \$(command -v podman)) 2>/dev/null || rpm -qf \$(readlink -f \$(command -v podman)) 2>/dev/null || echo 'not owned by dpkg/rpm (manual, static, or brew install)'"
  run "file -b \$(readlink -f \$(command -v podman)) | cut -c1-80"
  # Build tags, because a source build missing the `systemd` tag disables
  # container healthchecks outright and reports nothing at runtime.
  printf '$ go version -m $(command -v podman) | grep -- -tags=\n'
  printf '  %s\n' "$(podman_build_tags || echo '(unknown: no Go toolchain, stripped binary, or no -tags used)')"

  sec "network helpers"
  run "podman info --format 'netavark: {{.Host.NetworkBackendInfo.Path}} {{.Host.NetworkBackendInfo.Version}}'"
  run "podman info --format 'aardvark: {{.Host.NetworkBackendInfo.DNS.Path}} {{.Host.NetworkBackendInfo.DNS.Version}}'"
  run "ls -la /usr/lib/podman /usr/local/libexec/podman 2>/dev/null"

  sec "containers.conf / storage.conf / policy.json (full contents)"
  local f
  while read -r f; do
    printf '\n--- %s ---\n' "${f}"
    grep -vE '^[[:space:]]*(#|$)' "${f}" | sed 's/^/  /'
  done < <(containers_conf_files)
  for f in /etc/containers/storage.conf "${HOME}/.config/containers/storage.conf" \
           /etc/containers/policy.json "${HOME}/.config/containers/policy.json" \
           /usr/share/containers/policy.json; do
    [ -f "${f}" ] || continue
    printf '\n--- %s ---\n' "${f}"
    grep -vE '^[[:space:]]*(#|$)' "${f}" | sed 's/^/  /'
  done

  sec "cgroups"
  run "podman info --format 'cgroupManager={{.Host.CgroupManager}} version={{.Host.CgroupsVersion}}'"
  run "docker info --format 'service cgroupDriver={{.CgroupDriver}}' 2>/dev/null || echo 'docker CLI not usable'"
  run "cat /proc/self/cgroup"
  run "cat /sys/fs/cgroup/user.slice/user-\$(id -u).slice/user@\$(id -u).service/cgroup.controllers"

  sec "systemd user session"
  run "loginctl show-user ${u} --property=Linger --property=State"
  run "ls -la /run/user/\$(id -u)/bus"
  run "systemctl --user is-active podman.socket podman.service"
  run "systemctl --user show podman.service -p Environment -p ExecStart"
  run "journalctl --user -u podman.service --no-pager --since '-30min' | tail -25"

  sec "security / userns (Ubuntu 24.04+ restricts these)"
  run "sysctl kernel.apparmor_restrict_unprivileged_userns kernel.unprivileged_userns_clone user.max_user_namespaces net.ipv4.ip_unprivileged_port_start 2>/dev/null"
  run "ls /etc/apparmor.d/ 2>/dev/null | grep -iE 'podman|netavark|crun|rootlesskit|unshare' || echo '(no container apparmor profiles)'"
  run "grep -E \"^(${u}|root):\" /etc/subuid /etc/subgid"
  run "command -v newuidmap newgidmap || echo 'MISSING: install the uidmap package'"
  run "findmnt -no PROPAGATION /"
  run "grep -i microsoft /proc/version || echo '(not WSL)'"

  sec "docker front end"
  run "docker context ls"
  run "docker buildx version"
  run "systemctl is-active docker.service; systemctl --user is-active docker.service"

  sec "storage"
  run "podman info --format 'driver={{.Store.GraphDriverName}} nativeOverlayDiff={{index .Store.GraphStatus \"Native Overlay Diff\"}}'"

  # DDEV's own provider check. Worth having in full because it exercises the
  # paths this script only infers about -- in particular it runs a trivial
  # buildx build, which is the operation a cgroupfs cgroup manager actually
  # breaks. If that build succeeds, a cgroupfs finding is not the blocker.
  sec "ddev debug dockercheck"
  if ! command -v ddev >/dev/null 2>&1; then
    printf '  ddev is not on PATH; skipped\n'
    DOCKERCHECK_STATUS="ddev not installed"
    DOCKERCHECK_BUILD="unknown"
  elif [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    printf '  skipped (PODMAN_SKIP_SMOKE_TEST=true)\n'
    DOCKERCHECK_STATUS="skipped"
    DOCKERCHECK_BUILD="skipped"
  else
    printf '  (starts containers and runs a trivial build; may take a minute)\n'
    printf '$ ddev debug dockercheck\n'
    local dc_out dc_rc esc
    esc=$'\033'
    dc_out="$(ddev debug dockercheck 2>&1)"; dc_rc=$?
    # Strip colour so the report pastes cleanly into an issue.
    printf '%s\n' "${dc_out}" | sed -e "s/${esc}\[[0-9;]*m//g" -e 's/^/  /'
    printf '  [exit %s]\n' "${dc_rc}"
    [ "${dc_rc}" -eq 0 ] && DOCKERCHECK_STATUS="pass" || DOCKERCHECK_STATUS="FAIL"
    if printf '%s' "${dc_out}" | grep -qi 'buildx is working correctly'; then
      DOCKERCHECK_BUILD="ok"
    else
      DOCKERCHECK_BUILD="did NOT confirm a working buildx build"
    fi
  fi

  # Exercise the load path, not just record configuration. The report used to
  # print policy.json and leave a human to notice the consequence.
  sec "image load probe"
  check_image_load

  # Same reason as the load probe: watch a healthcheck actually run rather than
  # recording that one is configured. This is the check that a build missing
  # podman's systemd tag fails, and nothing else here notices that.
  sec "healthcheck probe"
  check_healthcheck
  run "systemctl --user list-timers --all --no-pager | tail -5"

  report_summary
  printf '\n===== end of report =====\n'
}

# The dump above is a wall of text. Distil it into the handful of facts a
# maintainer scans first, then call out anything that looks wrong, so the
# report can be triaged without reading all of it.
report_summary() {
  local kv notable=()
  kv() { printf '  %-18s %s\n' "$1" "$2"; }

  local distro kernel ppath pver porigin ppkg
  distro="$( . /etc/os-release 2>/dev/null && printf '%s %s (%s%s)' \
            "${NAME:-?}" "${VERSION_ID:-?}" "${ID:-?}" \
            "${ID_LIKE:+, like ${ID_LIKE}}" )"
  kernel="$(uname -srm)"
  ppath="$(command -v podman 2>/dev/null || echo '<none>')"
  pver="$(podman version --format '{{.Client.Version}}' 2>/dev/null || echo '?')"
  porigin="$(podman version --format '{{.Client.BuildOrigin}}' 2>/dev/null)"
  if dpkg -S "$(readlink -f "${ppath}")" >/dev/null 2>&1 ||
     rpm -qf "$(readlink -f "${ppath}")" >/dev/null 2>&1; then
    ppkg="distro-packaged"
  else
    ppkg="not distro-packaged"
  fi

  local nv nvpath av bundled cli_cg svc_cg linger bus store native ports userns ctx bx
  local ocirt ocipath
  # One podman call for everything. If podman itself is broken, the error text
  # is the single most useful thing in the report, so keep it and skip the
  # derived checks rather than printing a screenful of "?" and false findings.
  local info_ok=1 info_err="" infoline
  if infoline="$(podman info --format \
      '{{.Host.NetworkBackendInfo.Version}}|{{.Host.NetworkBackendInfo.Path}}|{{.Host.NetworkBackendInfo.DNS.Version}}|{{.Host.CgroupManager}}|{{.Store.GraphDriverName}}|{{index .Store.GraphStatus "Native Overlay Diff"}}|{{.Host.OCIRuntime.Name}}|{{.Host.OCIRuntime.Path}}' 2>&1)"; then
    IFS='|' read -r nv nvpath av cli_cg store native ocirt ocipath <<< "${infoline}"
    nv="${nv##* }"; av="${av##* }"
  else
    info_ok=0
    info_err="$(printf '%s' "${infoline}" | tail -1)"
    nv="unavailable"; av="unavailable"; cli_cg="unavailable"
    store="unavailable"; native="unavailable"; nvpath=""
    ocirt="unavailable"; ocipath=""
  fi
  bundled="$(bundled_helper_path netavark 2>/dev/null || true)"
  # Must collapse to one line: a failing docker info can emit blank lines,
  # which would otherwise wrap the summary and fake a cli/service mismatch.
  svc_cg="$(docker info --format '{{.CgroupDriver}}' 2>/dev/null | tr -d '\r' | grep -v '^$' | tail -1)"
  [ -z "${svc_cg}" ] && svc_cg="?"
  linger="$(loginctl show-user "$(id -un)" --property=Linger 2>/dev/null | cut -d= -f2)"
  [ -S "/run/user/$(id -u)/bus" ] && bus=present || bus=MISSING
  store="$(podman info --format '{{.Store.GraphDriverName}}' 2>/dev/null)"
  native="$(podman info --format '{{index .Store.GraphStatus "Native Overlay Diff"}}' 2>/dev/null)"
  ports="$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null)"
  userns="$(sysctl -n kernel.apparmor_restrict_unprivileged_userns 2>/dev/null || echo 'n/a')"
  ctx="$(docker context show 2>/dev/null)"
  bx="$(docker buildx version 2>/dev/null | awk '{print $2}')"

  sec "summary"
  kv "distro"        "${distro}"
  kv "kernel"        "${kernel}"
  kv "podman"        "${pver}${porigin:+ origin=${porigin}} (${ppkg}) ${ppath}"
  kv "netavark"      "${nv:-?} / aardvark ${av:-?}"
  kv "oci runtime"   "${ocirt:-?} ${ocipath}"
  kv "cgroup mgr"    "cli=${cli_cg} service=${svc_cg}"
  kv "linger / bus"  "${linger:-?} / ${bus}"
  kv "storage"       "${store:-?} (native overlay diff: ${native:-?})"
  kv "low ports"     "ip_unprivileged_port_start=${ports:-?}"
  kv "userns"        "apparmor_restrict_unprivileged_userns=${userns}"
  kv "docker ctx"    "${ctx:-?} (buildx ${bx:-none})"
  kv "dockercheck"   "${DOCKERCHECK_STATUS} (trivial build: ${DOCKERCHECK_BUILD})"
  kv "image load"    "${IMAGELOAD_STATUS}"
  kv "healthchecks"  "${HEALTHCHECK_STATUS}"
  kv "podman -tags"  "$(podman_build_tags || echo 'unknown')"
  kv "policy.json"   "$(active_policy_file 2>/dev/null || echo 'none -- pulls will fail with "no policy.json file found"')"

  # Anything worth looking at, most likely to be the cause first.
  local pmaj nmaj f
  pmaj="${pver%%.*}"; nmaj="${nv%%.*}"
  if [ "${info_ok}" -eq 0 ]; then
    notable+=("podman info FAILS, so podman is unusable: ${info_err}")
  fi
  if [ "${info_ok}" -eq 1 ] && [ -n "${pmaj}" ] && [ -n "${nmaj}" ] 2>/dev/null; then
    { [ "${pmaj}" -ge 6 ] && [ "${nmaj}" -lt 2 ]; } 2>/dev/null &&
      notable+=("podman ${pmaj}.x with netavark ${nmaj}.x -- version skew, no container will start")
    { [ "${pmaj}" -le 5 ] && [ "${nmaj}" -ge 2 ]; } 2>/dev/null &&
      notable+=("podman ${pmaj}.x with netavark ${nmaj}.x -- version skew")
  fi
  [ "${info_ok}" -eq 1 ] && [ -n "${bundled}" ] && [ -x "${bundled}" ] && [ -n "${nvpath}" ] &&
    [ "$(readlink -f "${bundled}")" != "$(readlink -f "${nvpath}")" ] &&
    notable+=("netavark in use is not the one podman shipped with (${nvpath})")
  if [ "${info_ok}" -eq 1 ] && [ "${cli_cg}" != "systemd" ]; then
    if [ "${DOCKERCHECK_BUILD}" = "ok" ]; then
      notable+=("cgroup manager is ${cli_cg} rather than systemd, but buildx built successfully anyway, so this is probably NOT the blocker")
    else
      notable+=("cgroup manager is ${cli_cg}, not systemd -- this can break compose builds")
    fi
  fi
  [ "${info_ok}" -eq 1 ] && [ "${cli_cg}" != "${svc_cg}" ] && [ "${svc_cg}" != "?" ] &&
    notable+=("cli and service disagree on cgroup manager (${cli_cg} vs ${svc_cg}); builds follow the service")
  while read -r f; do
    grep -qE '^[[:space:]]*helper_binaries_dir[[:space:]]*=' "${f}" 2>/dev/null &&
      notable+=("helper_binaries_dir set in ${f}")
    grep -qE '^[[:space:]]*cgroup_manager[[:space:]]*=' "${f}" 2>/dev/null &&
      notable+=("cgroup_manager set explicitly in ${f}")
  done < <(containers_conf_files)
  [ "${linger}" != "yes" ] && notable+=("systemd lingering is off")
  [ "${bus}" = "MISSING" ] && notable+=("no systemd user session bus")
  ! [ "${ports:-99999}" -le 80 ] 2>/dev/null &&
    notable+=("ports 80/443 not bindable (ip_unprivileged_port_start=${ports})")
  [ "${native}" = "false" ] && notable+=("native overlay diff off (fuse-overlayfs); slower than necessary")
  systemctl is-active docker.service >/dev/null 2>&1 &&
    notable+=("rootful docker.service running alongside podman")
  systemctl --user is-active docker.service >/dev/null 2>&1 &&
    notable+=("rootless docker.service running alongside podman")
  [ -z "${bx}" ] && notable+=("no docker buildx plugin; ddev start will fail at the build step")
  active_policy_file >/dev/null 2>&1 ||
    notable+=("no policy.json anywhere podman searches; pulls fail with \"no policy.json file found\"")
  local polgap; polgap="$(policy_archive_gap 2>/dev/null || true)"
  [ -n "${polgap}" ] &&
    notable+=("$(active_policy_file) rejects by default and allows no ${polgap}, so buildx --load is refused: builds succeed and then fail at the final image-load step")
  case "${IMAGELOAD_STATUS}" in
    FAIL*) notable+=("a built image cannot be loaded into the engine: ${IMAGELOAD_STATUS}") ;;
  esac
  case "${HEALTHCHECK_STATUS}" in
    *"systemd build tag"*)
      notable+=("podman was built without its 'systemd' build tag, so healthcheck timers are compiled as no-ops: no container can ever report healthy and every 'ddev start' times out") ;;
    FAIL*)
      notable+=("container healthchecks never report healthy (${HEALTHCHECK_STATUS}), so 'ddev start' cannot finish") ;;
  esac
  local btags; btags="$(podman_build_tags 2>/dev/null || true)"
  [ -n "${btags}" ] && ! printf '%s' "${btags}" | tr ',' '\n' | grep -qx systemd &&
    notable+=("podman build tags do not include 'systemd' (-tags=${btags})")
  [ "${DOCKERCHECK_STATUS}" = "FAIL" ] &&
    notable+=("ddev debug dockercheck failed -- its output above is the most direct evidence")

  printf '\n  notable:\n'
  if [ "${#notable[@]}" -eq 0 ]; then
    printf '    nothing obviously wrong\n'
  else
    printf '    - %s\n' "${notable[@]}"
  fi
}

run_checks() {
  # Checks must all run; do not abort on the first failure.
  set +o errexit
  printf 'Validating rootless Podman setup for DDEV\n'

  if check_podman_present; then
    check_socket
    check_service_path
    check_helper_versions
    check_oci_runtime
    check_helper_binaries_dir
    check_policy
    check_cgroups
    check_storage
  fi
  check_ports
  check_mount_propagation
  check_subuid
  check_docker_cli
  check_buildx
  check_smoke_test
  check_image_load
  check_healthcheck
  set -o errexit

  printf '\n'
  if [ "${problems}" -gt 0 ]; then
    printf '%s%d problem(s), %d warning(s)%s\n' "${red}" "${problems}" "${warnings}" "${reset}"
    return 1
  fi
  if [ "${warnings}" -gt 0 ]; then
    printf '%s%d warning(s), no blocking problems%s\n' "${yellow}" "${warnings}" "${reset}"
    return 0
  fi
  printf '%sAll checks passed%s\n' "${green}" "${reset}"
  return 0
}

# ---------------------------------------------------------------------------

main() {
  case "${1:-}" in
    --check|-c)
      run_checks
      ;;
    --report|-r)
      do_report
      ;;
    --help|-h)
      usage
      ;;
    "")
      do_install
      run_checks
      ;;
    *)
      printf 'Unknown argument: %s\n\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
}

main "$@"
