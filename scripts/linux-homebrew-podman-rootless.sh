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
#   PODMAN_DNS_SERVERS           Comma-separated DNS servers to force for
#                                containers, e.g. "1.1.1.1,1.0.0.1". Empty
#                                (default) leaves Podman's DNS handling alone.
#   PODMAN_USE_FUSE_OVERLAYFS    "true" to force the fuse-overlayfs mount
#                                program. Default "false", which lets Podman
#                                use native rootless overlay. Native overlay
#                                is available on kernel 5.13+ and is faster;
#                                fuse-overlayfs disables native overlay diff.
#   PODMAN_SKIP_SMOKE_TEST       "true" to skip the run-a-real-container check.

set -o errexit
set -o pipefail
set -o nounset

PODMAN_DNS_SERVERS="${PODMAN_DNS_SERVERS:-}"
PODMAN_USE_FUSE_OVERLAYFS="${PODMAN_USE_FUSE_OVERLAYFS:-false}"
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

# ---------------------------------------------------------------------------
# Install / configure
# ---------------------------------------------------------------------------

do_install() {
  heading "Installing rootless Podman (Homebrew)"

  # Stop any rootful Docker so it cannot compete for the socket or for ports.
  sudo systemctl disable --now docker.service docker.socket 2>/dev/null || true
  sudo rm -f /var/run/docker.sock

  # Remove the distro container stack. netavark and aardvark-dns MUST go too:
  # apt leaves them behind when podman is removed, and a stale
  # /usr/lib/podman/netavark is exactly what breaks a newer Homebrew podman.
  sudo apt-get remove -y podman crun netavark aardvark-dns 2>/dev/null || true

  if [ "${PODMAN_USE_FUSE_OVERLAYFS}" = "true" ]; then
    sudo apt-get install -y fuse-overlayfs
  fi

  brew install podman >/dev/null
  hash -r

  # Allow binding ports below 1024 so the DDEV router can use 80/443.
  sudo mkdir -p /etc/sysctl.d
  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee /etc/sysctl.d/60-rootless.conf >/dev/null
  sudo sysctl -p /etc/sysctl.d/60-rootless.conf

  # Without a genuine lingering session, systemd --user has no D-Bus session
  # bus to manage cgroups through, so podman silently falls back to
  # --cgroup-manager=cgroupfs. buildx then pins its buildkit container to the
  # "/docker/buildx" cgroup parent whenever podman reports "cgroupfs" as its
  # driver, which a rootless user can't create outside its own delegated
  # subtree, and every docker-compose build fails with:
  #   crun: create `/sys/fs/cgroup/docker`: Permission denied: OCI permission denied
  # See https://github.com/containers/podman/issues/5443 and
  # https://github.com/containers/podman/pull/29303.
  sudo loginctl enable-linger "$(whoami)"
  for _ in $(seq 1 10); do
    [ -S "/run/user/$(id -u)/bus" ] && break
    sleep 1
  done

  mkdir -p ~/.config/systemd/user
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
ExecStart=${BREW_PREFIX}/bin/podman \$LOGGING system service

[Install]
WantedBy=default.target
EOF

  # Fix for Podman 6: "registries.conf must be in v2 format but is in v1"
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
    if [ -n "${PODMAN_DNS_SERVERS}" ]; then
      printf 'dns_servers = ['
      printf '"%s", ' ${PODMAN_DNS_SERVERS//,/ } | sed 's/, $//'
      printf ']\n'
    fi
    echo 'log_driver = "k8s-file"'
    echo
    echo '[engine]'
    echo 'events_logger = "file"'
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

  # https://github.com/containers/podman/blob/main/docs/tutorials/performance.md#choosing-a-storage-driver
  mkdir -p ~/.config/containers
  if [ "${PODMAN_USE_FUSE_OVERLAYFS}" = "true" ]; then
    cat > ~/.config/containers/storage.conf <<'EOF'
[storage]
driver = "overlay"
[storage.options.overlay]
mount_program = "/usr/bin/fuse-overlayfs"
EOF
  else
    # Native rootless overlay (kernel 5.13+) keeps "Native Overlay Diff" on,
    # which fuse-overlayfs turns off.
    cat > ~/.config/containers/storage.conf <<'EOF'
[storage]
driver = "overlay"
EOF
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
    # A run of failed activations (a broken netavark makes podman exit 125 on
    # every connection) trips the systemd start limit and leaves both units in
    # "failed". Fixing the underlying config does NOT clear that by itself:
    # without reset-failed, `start` refuses and the stale socket file makes the
    # Docker CLI report "Cannot connect to the Docker daemon".
    if systemctl --user is-failed podman.socket >/dev/null 2>&1 ||
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

  # The loaded gun: a distro netavark sitting where a newer podman can find it.
  if [ -e /usr/lib/podman/netavark ] &&
     [ "$(readlink -f "${resolved_netavark}")" != "$(readlink -f /usr/lib/podman/netavark)" ]; then
    warn "an unused distro netavark remains at /usr/lib/podman/netavark"
    hint "Not currently in use, but any helper_binaries_dir change can select it."
    hint "sudo apt-get purge netavark aardvark-dns"
  fi
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
    hint "Remove that setting; podman picks systemd on its own when it can."
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

check_subuid() {
  heading "subuid / subgid"
  local user f
  user="$(id -un)"
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
    hint "The Docker CLI hides the real error. See the server side with:"
    hint "  journalctl --user -u podman.service --since '-5min'"
  fi
}

# buildx builds an image and then hands the result back to the container engine
# to load. That final load is a separate operation with its own failure mode:
# containers/image applies the signature policy to it, and a policy that does
# not accept the docker-archive transport rejects it. The build succeeds in
# full and only the last step fails, with:
#   failed to load image: payload does not match any of the supported image
#   formats: ... Source image rejected: ... is rejected by policy.
# Nothing else in this script exercises that path, and a plain `podman run`
# never touches it, so a machine can pull, run and build yet still be unable
# to finish `ddev start`. Probe it with an empty image: no network, ~4KB.
check_image_load() {
  heading "Image load (the step after a build)"
  if [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    ok "skipped (PODMAN_SKIP_SMOKE_TEST=true)"
    return 0
  fi

  local tag="ddev-policy-probe" ctx tar out
  ctx="$(mktemp -d)"
  tar="$(mktemp -u)".tar

  # A signature policy can reject at any of build, save or load, so treat a
  # rejection anywhere as the same finding rather than skipping the check.
  _probe_failed() {
    local step="$1" msg="$2"
    if printf '%s' "${msg}" | grep -q 'rejected by policy'; then
      fail "the image signature policy rejects images (at ${step})"
      hint "This is what makes 'ddev start' fail at the very end, after the"
      hint "build has already succeeded, with:"
      hint "  failed to load image: ... Source image rejected: ... rejected by policy"
      hint "Check the default in whichever of these exists:"
      hint "  ${HOME}/.config/containers/policy.json   (takes precedence)"
      hint "  /etc/containers/policy.json"
      hint "DDEV needs a permissive default:"
      hint '  {"default": [{"type": "insecureAcceptAnything"}]}'
    else
      warn "could not ${step} the probe image; skipping the load check"
      hint "$(printf '%s' "${msg}" | tail -2)"
    fi
  }

  if ! out="$(printf 'FROM scratch\n' | podman build -q -t "${tag}" -f - "${ctx}" 2>&1)"; then
    _probe_failed "build" "${out}"
  elif ! out="$(podman save -o "${tar}" "${tag}" 2>&1)"; then
    _probe_failed "save" "${out}"
  elif ! out="$(podman load -i "${tar}" 2>&1)"; then
    _probe_failed "load" "${out}"
  else
    ok "a built image can be saved and loaded back into podman"
  fi

  podman rmi -f "${tag}" >/dev/null 2>&1 || true
  rm -rf "${ctx}" "${tar}"
  unset -f _probe_failed
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
           /etc/containers/policy.json "${HOME}/.config/containers/policy.json"; do
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
  # One podman call for everything. If podman itself is broken, the error text
  # is the single most useful thing in the report, so keep it and skip the
  # derived checks rather than printing a screenful of "?" and false findings.
  local info_ok=1 info_err="" infoline
  if infoline="$(podman info --format \
      '{{.Host.NetworkBackendInfo.Version}}|{{.Host.NetworkBackendInfo.Path}}|{{.Host.NetworkBackendInfo.DNS.Version}}|{{.Host.CgroupManager}}|{{.Store.GraphDriverName}}|{{index .Store.GraphStatus "Native Overlay Diff"}}' 2>&1)"; then
    IFS='|' read -r nv nvpath av cli_cg store native <<< "${infoline}"
    nv="${nv##* }"; av="${av##* }"
  else
    info_ok=0
    info_err="$(printf '%s' "${infoline}" | tail -1)"
    nv="unavailable"; av="unavailable"; cli_cg="unavailable"
    store="unavailable"; native="unavailable"; nvpath=""
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
  kv "cgroup mgr"    "cli=${cli_cg} service=${svc_cg}"
  kv "linger / bus"  "${linger:-?} / ${bus}"
  kv "storage"       "${store:-?} (native overlay diff: ${native:-?})"
  kv "low ports"     "ip_unprivileged_port_start=${ports:-?}"
  kv "userns"        "apparmor_restrict_unprivileged_userns=${userns}"
  kv "docker ctx"    "${ctx:-?} (buildx ${bx:-none})"
  kv "dockercheck"   "${DOCKERCHECK_STATUS} (trivial build: ${DOCKERCHECK_BUILD})"

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
    check_helper_versions
    check_helper_binaries_dir
    check_cgroups
    check_storage
  fi
  check_ports
  check_subuid
  check_docker_cli
  check_buildx
  check_smoke_test
  check_image_load
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
