#!/usr/bin/env bash

# Set up rootless Podman as a DDEV container provider on Linux, with the Docker
# CLI as its front end, and validate the result.
#
# Usage:
#   linux-podman-rootless.sh                 Validate only; change nothing
#   linux-podman-rootless.sh --check         The same, said explicitly
#   linux-podman-rootless.sh --install       Install, configure, then validate
#   linux-podman-rootless.sh --install-brew  Same, with Podman from Homebrew
#   linux-podman-rootless.sh --report        Dump the environment for a bug report
#   linux-podman-rootless.sh --help          Show this help
#
# Validating is the default because installing prompts for sudo and writes files
# under /etc and ~/.config.
#
# --install takes Podman from the distribution's package manager, which is what
# you want. --install-brew is a last resort for when that Podman is too old
# (Ubuntu 24.04 ships 4.9.3): it *removes* the distribution's podman, crun,
# netavark and aardvark-dns first, because a stack half from the distribution and
# half from Homebrew starts no containers at all.
#
# --check and --report change nothing, work against any Podman install and read
# no secrets. --check exits nonzero on a problem; --report dumps distro, Podman
# provenance, every containers.conf, cgroup and systemd state and userns sysctls
# in one paste-able block.
#
# Symptom, cause and fix for everything checked here:
# https://docs.ddev.com/en/stable/users/install/docker-installation/#podman-rootless-troubleshooting
#
# Environment variables:
#   Name                    Default                     Effect when set
#   PODMAN_SKIP_SMOKE_TEST  false                       true skips the three checks that start
#                                                       a real container -- smoke test, image
#                                                       load, healthcheck -- which are the slow
#                                                       ones, and the only proof the setup works
#   BREW_PREFIX             /home/linuxbrew/.linuxbrew  Where a Homebrew podman, and the
#                                                       policy.json it ships, are looked for
#   SMOKE_TEST_IMAGE        ddev/ddev-utilities:latest  Image the container probes run
#   SUBID_RANGE             100000-165535               Range --install grants this user, shifted
#                                                       up when another user already holds it
#   CI                      unset                       true lets --install do what is only safe
#                                                       on a throwaway runner: disable rootful
#                                                       Docker, replace a v1 registries.conf, and
#                                                       autoremove podman's leftovers when
#                                                       --install-brew removes it (GA sets this)

set -o errexit
set -o pipefail
set -o nounset

PODMAN_SKIP_SMOKE_TEST="${PODMAN_SKIP_SMOKE_TEST:-false}"

BREW_PREFIX="${BREW_PREFIX:-/home/linuxbrew/.linuxbrew}"
SMOKE_TEST_IMAGE="${SMOKE_TEST_IMAGE:-ddev/ddev-utilities:latest}"
DOCKER_CONTEXT_NAME="podman-rootless"
SYSCTL_CONF="/etc/sysctl.d/60-ddev-podman-rootless.conf"
SUBID_RANGE="${SUBID_RANGE:-100000-165535}"

USE_BREW=false

problems=0
warnings=0

# Filled in by the probes, so report_summary can quote them without re-running.
DOCKERCHECK_STATUS="not run"
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
# Announce a step before running it, so a sudo prompt never comes out of nowhere.
step()    { printf '  -> %s\n' "$*"; }

# Print the header comment block, so --help never drifts from the file.
usage() {
  awk 'NR < 3 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

# ---------------------------------------------------------------------------
# Facts about this machine's podman
# ---------------------------------------------------------------------------

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

# Directory holding the podman on PATH, which is also where its conmon and crun
# live. Not resolved through symlinks on purpose: Homebrew's bin/podman points
# into Cellar/podman, where the tools of *other* formulae are not.
podman_bin_dir() {
  local podman_bin
  podman_bin="$(command -v podman 2>/dev/null)" || return 1
  dirname "${podman_bin}"
}

podman_is_brew() {
  case "$(command -v podman 2>/dev/null)" in
    "${BREW_PREFIX}"/*) return 0 ;;
    *) return 1 ;;
  esac
}

# The netavark/aardvark-dns that ship alongside the podman binary. They are
# versioned with podman -- 6.x needs netavark 2.x, 5.x needs 1.x -- and skew
# breaks every container start.
bundled_helper_path() {
  local helper="$1" podman_bin
  podman_bin="$(command -v podman 2>/dev/null)" || return 1
  podman_bin="$(readlink -f "${podman_bin}")"
  printf '%s/libexec/podman/%s' "$(dirname "$(dirname "${podman_bin}")")" "${helper}"
}

# The Go build tags podman was compiled with, which `podman version` does not
# report. One of them, `systemd`, silently decides whether healthchecks work at
# all (see check_healthcheck). Nonzero when the answer is unknown -- no Go
# toolchain, a stripped binary (Homebrew's is), or no -tags -- rather than guessing.
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

# True when podman would pick the systemd cgroup manager by itself. Asking for
# systemd where any of the three is missing (cgroup v1 host, WSL2 without
# systemd) is worse than the cgroupfs fallback: podman then creates no
# containers at all.
systemd_cgroups_available() {
  [ -f /sys/fs/cgroup/cgroup.controllers ] &&
    [ -d /run/systemd/system ] &&
    [ -S "/run/user/$(id -u)/bus" ]
}

# Whether a containers.conf we do not own sets a key. Ours is excluded on
# purpose: an override is only worth writing when something else sets the key,
# and matching our own output would make the override vanish on a second run.
system_conf_sets() {
  local key="$1" f
  for f in /usr/share/containers/containers.conf \
           /etc/containers/containers.conf \
           /etc/containers/containers.conf.d/*.conf; do
    [ -f "${f}" ] || continue
    grep -Eq "^[[:space:]]*${key}[[:space:]]*=" "${f}" && return 0
  done
  return 1
}

# ---------------------------------------------------------------------------
# Install / configure
# ---------------------------------------------------------------------------

apt_updated=false

pkg_install() {
  if command -v apt-get >/dev/null 2>&1; then
    if [ "${apt_updated}" != "true" ]; then
      sudo apt-get update -qq
      apt_updated=true
    fi
    sudo apt-get install -y "$@"
  elif command -v dnf >/dev/null 2>&1; then
    sudo dnf install -y "$@"
  else
    fail "neither apt-get nor dnf found; install these yourself, then re-run: $*"
    exit 1
  fi
}

# Remove the distribution's podman *and* what it brought in: anything left under
# /usr/lib/podman, a stale netavark above all, is what the Homebrew podman then
# finds instead of its own. Only safe after ensure_uidmap_for_brew, or the
# autoremove below takes uidmap too.
prune_distro_podman_for_brew() {
  step "Removing the distribution's podman and its dependencies (sudo)"
  local pkg
  # One at a time: naming a package the distribution does not have fails the
  # whole command, and then nothing is removed at all.
  for pkg in podman crun netavark aardvark-dns; do
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get purge -y "${pkg}" 2>/dev/null || true
    elif command -v dnf >/dev/null 2>&1; then
      sudo dnf remove -y "${pkg}" 2>/dev/null || true
    fi
  done
  # Sweeps up the rest of what podman pulled in, but also anything else already
  # orphaned, which this script has no business removing off a runner.
  if [ "${CI:-}" = "true" ]; then
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get autoremove -y --purge 2>/dev/null || true
    elif command -v dnf >/dev/null 2>&1; then
      sudo dnf autoremove -y 2>/dev/null || true
    fi
  fi
}

# podman needs setuid newuidmap/newgidmap for the rootless user namespace, so
# they can only come from the distribution, and normally arrive with its podman
# package. The brew path removes that package, so claim them first: install if
# missing, and mark manual so no autoremove -- ours or a later one -- takes them.
ensure_uidmap_for_brew() {
  if ! command -v newuidmap >/dev/null 2>&1 || ! command -v newgidmap >/dev/null 2>&1; then
    step "Installing newuidmap/newgidmap (sudo)"
    if command -v apt-get >/dev/null 2>&1; then
      pkg_install uidmap
    else
      pkg_install shadow-utils
    fi
  fi
  if command -v apt-mark >/dev/null 2>&1; then
    step "Protecting uidmap from a future 'apt-get autoremove' (sudo)"
    sudo apt-mark manual uidmap 2>/dev/null || true
  fi
}

install_podman_from_distro() {
  if command -v podman >/dev/null 2>&1; then
    return 0
  fi
  # No uidmap handling: the distribution's podman package pulls it in itself, and
  # nothing in this path removes it again.
  step "Installing podman (sudo)"
  pkg_install podman
  hash -r
}

install_podman_from_brew() {
  if ! command -v brew >/dev/null 2>&1; then
    fail "--install-brew was given but brew is not on PATH; see https://brew.sh"
    exit 1
  fi
  ensure_uidmap_for_brew
  prune_distro_podman_for_brew

  step "brew install podman"
  brew install podman >/dev/null
  hash -r

  # GitHub Actions' runner image unpacks a static podman 5.x into /usr/local/bin,
  # and whichever directory comes first in PATH wins -- silently, because that
  # stack is self-consistent and passes --check.
  if ! podman_is_brew; then
    warn "podman resolves to $(command -v podman), not ${BREW_PREFIX}/bin/podman"
  fi

  # Homebrew's bottles use their own dynamic linker, which does not search the
  # host's /usr/lib/<arch>-linux-gnu paths. netavark and aardvark-dns need
  # libgcc_s.so.1 and nothing in podman's Homebrew dependencies provides it, so
  # without gcc every container start fails on "cannot open shared object file".
  local netavark_bin
  netavark_bin="$(bundled_helper_path netavark || true)"
  if [ -n "${netavark_bin}" ] && [ -x "${netavark_bin}" ] && ! "${netavark_bin}" --version >/dev/null 2>&1; then
    step "brew install gcc, for the libgcc_s.so.1 netavark needs"
    brew install gcc >/dev/null
    hash -r
  fi
}

# SUBID_RANGE itself when nothing in ${1} holds it, otherwise the first free block
# of the same size above it. usermod needs an explicit range, and two users given
# the same one map the same host IDs: whoever starts a container second breaks.
free_subid_range() {
  [ -f "$1" ] || { printf '%s\n' "${SUBID_RANGE}"; return 0; }
  awk -F: -v want="${SUBID_RANGE%%-*}" \
      -v count="$(( ${SUBID_RANGE##*-} - ${SUBID_RANGE%%-*} + 1 ))" '
    NF >= 3 { n++; s[n] = $2 + 0; e[n] = $2 + $3 }
    END {
      cand = want
      do {
        moved = 0
        for (i = 1; i <= n; i++)
          if (cand < e[i] && s[i] < cand + count) { cand = e[i]; moved = 1 }
      } while (moved)
      printf "%d-%d\n", cand, cand + count - 1
    }' "$1"
}

configure_subids() {
  local user range changed=0
  user="$(id -un)"
  if ! grep -q "^${user}:" /etc/subuid 2>/dev/null; then
    range="$(free_subid_range /etc/subuid)"
    step "Adding subuid range ${range} for ${user} (sudo)"
    [ "${range}" = "${SUBID_RANGE}" ] || hint "${SUBID_RANGE} is already in use"
    sudo usermod --add-subuids "${range}" "${user}"
    changed=1
  fi
  if ! grep -q "^${user}:" /etc/subgid 2>/dev/null; then
    range="$(free_subid_range /etc/subgid)"
    step "Adding subgid range ${range} for ${user} (sudo)"
    [ "${range}" = "${SUBID_RANGE}" ] || hint "${SUBID_RANGE} is already in use"
    sudo usermod --add-subgids "${range}" "${user}"
    changed=1
  fi
  # podman caches the ranges it started with, and keeps using the old (empty)
  # mapping without this.
  [ "${changed}" -eq 1 ] && podman system migrate >/dev/null 2>&1 || true
}

configure_sysctls() {
  step "Allowing low ports and unprivileged user namespaces (sudo)"
  sudo mkdir -p /etc/sysctl.d
  {
    # Our own file, so re-running overwrites nothing but our own previous values.
    # 60-rootless.conf, which the docs and CI append to, belongs to whoever else
    # wrote it.
    echo "# Written by ddev's linux-podman-rootless.sh"
    # So the DDEV router can bind 80/443.
    echo 'net.ipv4.ip_unprivileged_port_start=0'
    # Debian ships this switched off; Fedora ships max_user_namespaces at 0 on
    # some releases. Both stop rootless podman from creating a user namespace.
    if [ -f /proc/sys/kernel/unprivileged_userns_clone ]; then
      echo 'kernel.unprivileged_userns_clone=1'
    fi
    if [ "$(cat /proc/sys/user/max_user_namespaces 2>/dev/null || echo 1)" = "0" ]; then
      echo 'user.max_user_namespaces=28633'
    fi
  } | sudo tee "${SYSCTL_CONF}" >/dev/null
  sudo sysctl -p "${SYSCTL_CONF}"
}

# A machine with no session of its own -- headless, CI -- has no session bus, so
# podman falls back to the cgroupfs cgroup manager and compose builds break.
# Lingering creates the bus there. A desktop session already has one, hence the
# gate: this is not worth turning on for its own sake.
# https://github.com/containers/podman/issues/5443
# https://github.com/containers/podman/pull/29303
enable_linger() {
  if [ -S "/run/user/$(id -u)/bus" ]; then
    return 0
  fi
  step "Enabling systemd lingering, to give this user a session bus (sudo)"
  sudo loginctl enable-linger "$(whoami)"
  for _ in $(seq 1 10); do
    [ -S "/run/user/$(id -u)/bus" ] && break
    sleep 1
  done
}

# Homebrew does ship podman.socket/podman.service, but they carry no
# Environment=PATH, so the service inherits systemd --user's minimal PATH and
# cannot find conmon -- and the Cellar copy is read-only, so it cannot be fixed
# in place. Write our own with the PATH baked in.
write_user_units() {
  local podman_bin bin_dir
  podman_bin="$(command -v podman)"
  bin_dir="$(podman_bin_dir)"

  # A distro podman lives where the minimal PATH already reaches, so leave its
  # unit alone and let package updates keep it current.
  case "${bin_dir}" in
    /bin|/usr/bin|/usr/local/bin)
      if systemctl --user cat podman.service 2>/dev/null | grep -qF "ExecStart=${podman_bin}"; then
        return 0
      fi
      ;;
  esac

  step "Writing podman.socket and podman.service for ${podman_bin}"
  mkdir -p ~/.config/systemd/user
  # Either path may be a symlink into Homebrew's read-only Cellar, which `cat >`
  # would follow and fail on rather than replace.
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
# podman finds conmon/crun via PATH plus a fixed list of distro paths, and
# systemd --user hands it a minimal PATH. Without podman's own directory every
# container create fails with "could not find a working conmon binary".
Environment=PATH=${bin_dir}:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ExecStart=${podman_bin} \$LOGGING system service

[Install]
WantedBy=default.target
EOF
}

write_containers_conf() {
  local conf=~/.config/containers/containers.conf.d/ddev-podman.conf
  local bin_dir brew=0 cgroup=0 crun=0
  bin_dir="$(podman_bin_dir)"

  # Every setting below works around one specific broken thing. A distro podman on
  # a machine nothing else configured needs none of them, and gets no drop-in.
  if podman_is_brew; then
    brew=1
  fi
  # A system containers.conf pinning cgroupfs, which only a user drop-in outranks.
  # GitHub Actions' runner image does this, via the mgoltzsche/podman-static
  # bundle's own /etc.
  if systemd_cgroups_available && system_conf_sets cgroup_manager; then
    cgroup=1
  fi
  # Same bundle pins crun to the static /usr/local/bin/crun from its podman 5.x,
  # which an unrelated podman then uses.
  if [ -n "${bin_dir}" ] && [ -x "${bin_dir}/crun" ] && system_conf_sets crun; then
    crun=1
  fi

  if [ $((brew + cgroup + crun)) -eq 0 ]; then
    # Users are told to hand-edit this same file, so never remove it -- but a
    # leftover from an earlier setup still applies, so say it is there.
    if [ -f "${conf}" ]; then
      warn "${conf} exists, but nothing on this machine needs overriding"
      hint "It still applies to podman. Review it if podman misbehaves."
    fi
    return 0
  fi

  step "Writing ${conf}"
  mkdir -p ~/.config/containers/containers.conf.d
  {
    if [ "${brew}" -eq 1 ]; then
      # Homebrew's conmon is built without journald support, so podman's default
      # journald logging fails outright (containers/conmon#348).
      echo '[containers]'
      echo 'log_driver = "k8s-file"'
      echo
    fi

    echo '[engine]'
    if [ "${brew}" -eq 1 ]; then
      echo 'events_logger = "file"'
    fi
    if [ "${cgroup}" -eq 1 ]; then
      echo 'cgroup_manager = "systemd"'
    fi
    if [ "${crun}" -eq 1 ]; then
      echo
      echo '[engine.runtimes]'
      printf 'crun = ["%s/crun"]\n' "${bin_dir}"
    fi

    # Homebrew's netavark has dropped the iptables backend, while podman's
    # default still requests it in some version pairings ("Must provide a valid
    # firewall backend, got iptables"). Distro netavark 1.x predates nftables,
    # so only say this for Homebrew's.
    if [ "${brew}" -eq 1 ]; then
      echo
      echo '[network]'
      echo 'firewall_driver = "nftables"'
    fi
  } > "${conf}"

  # No engine.helper_binaries_dir on purpose: it *replaces* podman's helper
  # search path rather than extending it, so a list omitting podman's own libexec
  # directory selects a wrong-version netavark, or none at all.
}

# Only worth writing where something already turned the fast path off: native
# rootless overlay (kernel 5.13+) is podman's own default, and a fuse-overlayfs
# mount_program is what disables it.
# https://github.com/containers/podman/blob/main/docs/tutorials/performance.md#choosing-a-storage-driver
write_storage_conf() {
  local f
  for f in /usr/share/containers/storage.conf /etc/containers/storage.conf; do
    [ -f "${f}" ] || continue
    grep -Eq '^[[:space:]]*mount_program[[:space:]]*=' "${f}" || continue
    step "Overriding the ${f} fuse-overlayfs mount_program"
    mkdir -p ~/.config/containers
    cat > ~/.config/containers/storage.conf <<'EOF'
[storage]
driver = "overlay"
EOF
    return 0
  done
  return 0
}

install_policy_json() {
  # Without a policy.json anywhere podman searches, a pull fails outright with
  # "no policy.json file found". A distro podman ships one under /usr/share;
  # Homebrew's sits under its own prefix, which podman does not search.
  if ! active_policy_file >/dev/null; then
    step "Installing a policy.json"
    mkdir -p ~/.config/containers
    if [ -f "${BREW_PREFIX}/etc/containers/policy.json" ]; then
      cp "${BREW_PREFIX}/etc/containers/policy.json" ~/.config/containers/policy.json
    else
      printf '{"default": [{"type": "insecureAcceptAnything"}]}\n' > ~/.config/containers/policy.json
    fi
  fi
}

# Seen only on the GitHub Actions runner, whose preinstalled podman 5 leaves a v1
# registries.conf that a podman 6 over the top rejects outright. A drop-in cannot
# fix it, because podman rejects the base file, so it has to be replaced -- which
# throws away every registry configured in it. Only do that on a runner.
fix_registries_conf() {
  [ -f /etc/containers/registries.conf ] || return 0
  grep -q '^[[:space:]]*\[registries' /etc/containers/registries.conf || return 0
  local podman_major
  podman_major="$(podman version --format '{{.Client.Version}}' 2>/dev/null | cut -d. -f1)"
  [ "${podman_major:-0}" -ge 6 ] 2>/dev/null || return 0

  if [ "${CI:-}" != "true" ]; then
    warn "/etc/containers/registries.conf is v1, which podman ${podman_major} refuses to read"
    hint "Port your registries to the v2 format, or if you have none worth keeping:"
    hint '  echo "unqualified-search-registries = [\"docker.io\"]" | sudo tee /etc/containers/registries.conf'
    return 0
  fi

  step "Replacing the v1 /etc/containers/registries.conf that podman ${podman_major} rejects (sudo)"
  sudo cp /etc/containers/registries.conf /etc/containers/registries.conf.v1.bak
  sudo tee /etc/containers/registries.conf >/dev/null <<'EOF'
unqualified-search-registries = ["docker.io"]
EOF
  hint "previous file kept at /etc/containers/registries.conf.v1.bak"
}

point_docker_at_podman() {
  systemctl --user daemon-reload
  systemctl --user enable --now podman.socket

  # Can return "failed to reexec: Permission denied" on the first call.
  podman info --format '{{.Host.RemoteSocket.Path}}' >/dev/null 2>&1 ||
    podman info --format '{{.Host.RemoteSocket.Path}}' >/dev/null 2>&1 || true

  local sock
  sock="$(podman info --format '{{.Host.RemoteSocket.Path}}')"

  # DDEV drives podman through the Docker CLI, but the CLI and its buildx plugin
  # come from Docker's own repository, so this script does not install them.
  if ! command -v docker >/dev/null 2>&1; then
    warn "no docker CLI, so the '${DOCKER_CONTEXT_NAME}' context was not created"
    hint "Install the CLI and buildx (not the engine), then re-run:"
    hint "  sudo apt-get install -y docker-ce-cli docker-buildx-plugin"
    hint "  (or: sudo dnf install -y docker-ce-cli docker-buildx-plugin)"
    hint "Podman itself is set up and listening on ${sock}."
    return 0
  fi

  step "Pointing the docker context '${DOCKER_CONTEXT_NAME}' at ${sock}"
  docker context rm -f "${DOCKER_CONTEXT_NAME}" >/dev/null 2>&1 || true
  docker context create "${DOCKER_CONTEXT_NAME}" \
    --description "Podman (rootless)" \
    --docker host="unix://${sock}" >/dev/null
  docker context use "${DOCKER_CONTEXT_NAME}" >/dev/null
}

do_install() {
  if [ "${USE_BREW}" = "true" ]; then
    heading "Installing rootless Podman (Homebrew)"
  else
    heading "Installing rootless Podman"
  fi

  # CI only, so nothing can quietly reach the docker daemon instead of podman. A
  # workstation keeps its Docker: the two only compete for the router ports.
  if [ "${CI:-}" = "true" ]; then
    step "Stopping and disabling rootful Docker (sudo)"
    sudo systemctl disable --now docker.service docker.socket 2>/dev/null || true
    sudo rm -f /var/run/docker.sock
  fi

  if [ "${USE_BREW}" = "true" ]; then
    install_podman_from_brew
  else
    install_podman_from_distro
  fi

  # WSL2 mounts / before systemd starts and never marks it shared, so podman
  # warns on every container start and nested mounts can go missing. Lasts only
  # for this WSL2 instance; the docs cover persisting it via /etc/wsl.conf.
  if grep -qi microsoft /proc/version 2>/dev/null; then
    step "Making the WSL2 root filesystem a shared mount (sudo)"
    sudo mount --make-rshared /
  fi

  configure_subids
  configure_sysctls
  enable_linger
  write_user_units
  write_containers_conf
  write_storage_conf
  install_policy_json
  fix_registries_conf
  point_docker_at_podman
}

# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

check_podman_present() {
  heading "Podman"
  if ! command -v podman >/dev/null 2>&1; then
    fail "podman is not on PATH"
    hint "Run this script with --install, or --install-brew for a current one"
    hint "from Homebrew."
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
    if systemctl --user status podman.socket 2>&1 | grep -q 'could not be found'; then
      hint "systemctl --user does not know a podman.socket unit at all. A Homebrew"
      hint "podman keeps its units under \$(brew --prefix)/lib/systemd/user/, which"
      hint "systemctl --user never searches. Run this script with --install to"
      hint "write them into ~/.config/systemd/user/."
    # Repeated failed activations (a broken netavark makes podman exit 125 on
    # every connection) trip systemd's start limit, which fixing the underlying
    # config does not clear.
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

  # Lingering only matters where no logged-in session keeps systemd --user alive.
  # A desktop has the bus either way, and warning there sends people chasing a
  # non-problem.
  if loginctl show-user "$(whoami)" --property=Linger 2>/dev/null | grep -q 'Linger=yes'; then
    ok "systemd lingering is enabled"
  elif [ -S "/run/user/$(id -u)/bus" ]; then
    ok "no lingering, but this session has a user bus"
    hint "podman stops when your last session ends. To keep it running (needed for"
    hint "SSH-only or headless machines): sudo loginctl enable-linger $(whoami)"
  else
    warn "no lingering and no session bus for $(whoami)"
    hint "sudo loginctl enable-linger $(whoami)"
    hint "Without a session bus podman falls back to the cgroupfs cgroup manager;"
    hint "the cgroups check below says whether that actually happened."
  fi
}

# The breakage that costs the most time: podman resolving a network helper of a
# different major version than the one it shipped with.
check_helper_versions() {
  heading "Network helpers (netavark / aardvark-dns)"

  # Run the bundled binaries directly first. When they cannot execute at all,
  # podman info's NetworkBackendInfo fields come back empty, which says far less
  # than the loader's own error.
  local helper bundled errmsg f helper_broken=0
  for helper in netavark aardvark-dns; do
    bundled="$(bundled_helper_path "${helper}" || true)"
    [ -n "${bundled}" ] && [ -x "${bundled}" ] || continue
    if ! errmsg="$("${bundled}" --version 2>&1)"; then
      helper_broken=1
      fail "${bundled} fails to run: $(printf '%s' "${errmsg}" | tail -1)"
      if printf '%s' "${errmsg}" | grep -q 'libgcc_s'; then
        hint "Homebrew's dynamic linker does not search the host's library paths,"
        hint "and nothing in podman's Homebrew dependencies provides the"
        hint "libgcc_s.so.1 that ${helper} needs. Fix: brew install gcc"
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
    hint "Usually engine.helper_binaries_dir pointing at directories without a"
    hint "netavark: it REPLACES podman's search path instead of extending it."
    while read -r f; do
      grep -qE '^[[:space:]]*helper_binaries_dir[[:space:]]*=' "${f}" &&
        hint "set in ${f} -- remove it"
    done < <(containers_conf_files)
    return 0
  fi

  IFS='|' read -r resolved_netavark netavark_ver resolved_dns dns_ver <<< "${info}"

  for helper in netavark aardvark-dns; do
    local resolved ver
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
      hint "and remove the distribution's ${helper} package."
      continue
    fi

    ok "${ver} at ${resolved}"
  done

  # Explicit pairing check, for distro installs with no bundled helpers next to
  # the podman binary. Only skew is a problem, not the paths themselves.
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

}

# Same failure mode as the network helpers -- podman and crun are released as a
# pair -- but reached another way: a pin in someone else's containers.conf.
check_oci_runtime() {
  heading "OCI runtime"

  local resolved expected bin_dir
  resolved="$(podman info --format '{{.Host.OCIRuntime.Path}}' 2>/dev/null || true)"
  if [ -z "${resolved}" ]; then
    warn "podman did not report an OCI runtime path"
    return 0
  fi

  bin_dir="$(podman_bin_dir)"
  expected="${bin_dir}/$(basename "${resolved}")"

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

# An interactive shell finds conmon via PATH, but `systemd --user` services get a
# minimal one, so podman.service can fail with "could not find a working conmon
# binary" while `podman info` still reports the right path: that answer comes
# from the CLI's PATH, not the service's.
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
    hint "search list nor podman.service's PATH, so container create will fail."
    hint "Re-run this script with --install to rewrite the unit, or set it by hand"
    hint "under [Service]: Environment=PATH=${conmon_dir}:/usr/bin:/bin"
  fi
}

# Without one, a pull fails outright rather than falling back to any default:
# "config file not found: no policy.json file found; searched paths: [...]".
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
  local mgr svc
  mgr="$(podman info --format '{{.Host.CgroupManager}}' 2>/dev/null || echo unknown)"
  # Builds go through the service, so its answer is the one that counts, and it
  # can differ from the CLI's when the two started under different sessions.
  svc="$(docker info --format '{{.CgroupDriver}}' 2>/dev/null | grep -v '^$' | tail -1)"
  if [ "${mgr}" = "systemd" ]; then
    if [ -n "${svc}" ] && [ "${svc}" != "systemd" ]; then
      fail "the CLI uses systemd but podman.service uses ${svc}, and builds follow the service"
      hint "systemctl --user restart podman.socket"
      return 0
    fi
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

  # Work out why, in precedence order: lingering is only one of several causes,
  # and blaming it when it is already on sends people in circles.

  # 1. Explicitly configured. Wins over everything, so report it first.
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
    hint "which outranks everything in /etc. Running this script with --install"
    hint "writes that for you."
    return 0
  fi

  # 2. Whether podman is *able* to use the systemd manager. Running a container
  #    is the only real test: `podman --cgroup-manager=systemd info` just echoes
  #    the flag back.
  if [ "${PODMAN_SKIP_SMOKE_TEST}" != "true" ]; then
    local probe
    if probe="$(podman --cgroup-manager=systemd run --rm "${SMOKE_TEST_IMAGE}" true 2>&1)"; then
      hint "podman CAN run under the systemd manager here, so nothing is stopping"
      hint "it: its own default just picked cgroupfs, which normally means it saw"
      hint "no usable systemd user session at startup. A restart may be enough:"
      hint "  systemctl --user restart podman.socket"
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
  # The sysctl is the *lowest* port an unprivileged process may bind, so <= 80
  # covers both 80 and 443.
  if ! [ "${start}" -le 80 ] 2>/dev/null; then
    warn "net.ipv4.ip_unprivileged_port_start is ${start}, so ports 80/443 are unavailable"
    hint "Either allow low ports:"
    hint "  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee ${SYSCTL_CONF}"
    hint "  sudo sysctl -p ${SYSCTL_CONF}"
    hint "or configure DDEV for unprivileged ports:"
    hint "  ddev config global --router-http-port=8080 --router-https-port=8443"
  else
    ok "unprivileged processes may bind ports 80/443"
  fi
}

# WSL2-specific: podman warns on every container start and nested mounts can go
# missing. Not known to break typical DDEV usage, but cheap to flag and fix.
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

  # uidmap on Debian/Ubuntu, shadow-utils on Fedora. Without them podman cannot
  # set up the rootless user namespace at all.
  if ! command -v newuidmap >/dev/null 2>&1 || ! command -v newgidmap >/dev/null 2>&1; then
    fail "newuidmap/newgidmap not found"
    hint "sudo apt-get install -y uidmap   # Fedora: shadow-utils (usually already installed)"
  else
    ok "newuidmap/newgidmap present"
    # An auto-installed uidmap is only at risk once nothing installed depends on
    # it, which is what removing a distro podman does. With that podman still
    # there, apt keeps uidmap and this would be noise.
    if command -v apt-mark >/dev/null 2>&1 &&
       apt-mark showauto uidmap 2>/dev/null | grep -qx uidmap &&
       ! dpkg -S "$(readlink -f "$(command -v podman 2>/dev/null)")" >/dev/null 2>&1; then
      warn "uidmap is marked auto-installed, and no distro podman package holds it"
      hint "a future 'apt-get autoremove' could remove it and break rootless Podman"
      hint "fix: sudo apt-mark manual uidmap"
    fi
  fi

  for f in /etc/subuid /etc/subgid; do
    if ! grep -q "^${user}:" "${f}" 2>/dev/null; then
      fail "no entry for ${user} in ${f}"
      hint "sudo usermod --add-subuids $(free_subid_range /etc/subuid) \\"
      hint "               --add-subgids $(free_subid_range /etc/subgid) ${user}"
      continue
    fi
    # Overlapping ranges collide as soon as a second user (or rootful podman)
    # maps the same subordinate IDs. One user may hold several ranges, so
    # compare every range against every other.
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
  sock="$(podman info --format '{{.Host.RemoteSocket.Path}}' 2>/dev/null || true)"

  # DOCKER_HOST outranks the context, so check it first: with it set, the CLI can
  # be talking to another engine while `docker context show` still names this one.
  if [ -n "${DOCKER_HOST:-}" ]; then
    if [ -n "${sock}" ] && [ "${DOCKER_HOST}" = "unix://${sock}" ]; then
      ok "DOCKER_HOST -> ${DOCKER_HOST}"
    else
      fail "DOCKER_HOST is ${DOCKER_HOST}, and it overrides every docker context"
      hint "unset DOCKER_HOST, so the '${DOCKER_CONTEXT_NAME}' context applies"
    fi
  else
    ctx="$(docker context show 2>/dev/null || echo unknown)"
    endpoint="$(docker context inspect "${ctx}" --format '{{.Endpoints.docker.Host}}' 2>/dev/null || echo '')"
    if [ -n "${sock}" ] && [ "${endpoint}" = "unix://${sock}" ]; then
      ok "context '${ctx}' -> ${endpoint}"
    else
      fail "active docker context '${ctx}' points at ${endpoint:-nothing}, not unix://${sock}"
      hint "docker context use ${DOCKER_CONTEXT_NAME}"
    fi
  fi

  # The contexts, rather than guessed socket paths, are what the docker CLI can
  # actually reach. Running several engines at once is supported, but they
  # compete for the router ports.
  local name ep listed=0
  while read -r name ep; do
    case "${ep}" in unix://*) ;; *) continue ;; esac
    [ "${ep}" = "unix://${sock}" ] && continue
    [ -S "${ep#unix://}" ] || continue
    [ "${listed}" -eq 0 ] && warn "other engines are listening too:" && listed=1
    hint "${name} -> ${ep}"
  done < <(docker context ls --format '{{.Name}} {{.DockerEndpoint}}' 2>/dev/null)
  if [ "${listed}" -eq 1 ]; then
    hint "Only the active context is in use, but whichever engine holds a ddev"
    hint "router keeps the others from binding ports 80/443."
  fi
  return 0
}

# DDEV builds project images with `docker buildx`, and its default
# (docker_buildx_version: "system") downloads nothing. docker-ce-cli alone leaves
# buildx missing, and `ddev start` then fails at the build step, not at startup.
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
  # Exercises pull, create, network setup and attach -- the path a broken
  # netavark breaks.
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

# The load that follows a build applies the signature policy to a docker-archive,
# so a policy rejecting that transport fails `ddev start` after the build has
# fully succeeded. Nothing else here reaches that path: `podman run` loads no
# archive, and `ddev debug dockercheck` builds without --load.
check_image_load() {
  heading "Image load (the step after a build)"
  if [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    IMAGELOAD_STATUS="skipped"
    ok "skipped (PODMAN_SKIP_SMOKE_TEST=true)"
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1 || ! docker buildx version >/dev/null 2>&1; then
    IMAGELOAD_STATUS="skipped (no buildx)"
    warn "skipped, this needs the buildx plugin the check above asks for"
    return 0
  fi

  local tag="ddev-load-probe" ctx out
  ctx="$(mktemp -d)"
  if out="$(printf 'FROM scratch\n' | docker buildx build --load -f - -t "${tag}" "${ctx}" 2>&1)"; then
    IMAGELOAD_STATUS="ok"
    ok "buildx can build an image and load it into the engine"
  elif printf '%s' "${out}" | grep -q 'rejected by policy'; then
    IMAGELOAD_STATUS="FAIL (rejected by policy)"
    fail "the image signature policy rejects images"
    hint "This is what makes 'ddev start' fail at the very end, after the build"
    hint "has already succeeded, with:"
    hint "  failed to load image: ... Source image rejected: ... rejected by policy"
    hint "Fix it in $(active_policy_file): either allow the docker-archive and"
    hint "oci-archive transports buildx --load uses, or accept everything with"
    hint '  {"default": [{"type": "insecureAcceptAnything"}]}'
  else
    IMAGELOAD_STATUS="FAIL"
    fail "a built image could not be loaded into the engine"
    hint "$(printf '%s' "${out}" | tail -3)"
  fi

  docker rmi -f "${tag}" >/dev/null 2>&1 || true
  rm -rf "${ctx}"
}

# `ddev start` waits for each container's own HEALTHCHECK, so one that never runs
# stalls every start until the timeout, with an empty health log rather than a
# failing one. Podman schedules them as transient systemd *user* timers, so two
# things must hold beyond the command itself working: podman compiled with its
# `systemd` build tag (without it libpod's timer functions are silent no-ops), and
# a session that accepts transient units. Running the healthcheck by hand tests
# neither, so watch a real container reach `healthy` instead.
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
  # Through the Docker CLI: a healthcheck registered over the compat API is the
  # case that matters for DDEV.
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

    # Whether podman scheduled a timer at all separates the two failure shapes.
    local timer=no
    systemctl --user list-units --all --no-pager "${cid:0:32}*" 2>/dev/null |
      grep -q '\.timer' && timer=yes

    log="$(docker inspect --format '{{json .State.Health.Log}}' "${name}" 2>/dev/null || true)"
    case "${log}" in
      ''|null|'[]')
        hint "The health log is empty, so the command never ran or never returned;"
        hint "the command itself is not the problem." ;;
      *) hint "The healthcheck ran and failed: ${log}" ;;
    esac

    # A missing build tag fails this way unconditionally and silently, so rule it
    # in or out before looking at the systemd session.
    if [ "${timer}" = "no" ]; then
      hint "No healthcheck timer unit exists for the probe container, so podman"
      hint "never even tried to schedule one."
      if tags="$(podman_build_tags)" && ! printf '%s' "${tags}" | tr ',' '\n' | grep -qx systemd; then
        HEALTHCHECK_STATUS="FAIL (podman built without its systemd build tag)"
        hint "podman was compiled without its 'systemd' build tag (-tags=${tags}),"
        hint "which turns its healthcheck timers into no-ops. No configuration"
        hint "works around that; install a podman built with it."
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

# Everything needed to diagnose someone else's machine in one paste: --check says
# what looks wrong, --report says what is there.
do_report() {
  # A failing command is itself a datum, not a reason to stop the dump.
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
  # How podman was installed decides who owns the defaults under /usr/share.
  run "dpkg -S \$(readlink -f \$(command -v podman)) 2>/dev/null || rpm -qf \$(readlink -f \$(command -v podman)) 2>/dev/null || echo 'not owned by dpkg/rpm (manual, static, or brew install)'"
  run "file -b \$(readlink -f \$(command -v podman)) | cut -c1-80"
  # A source build missing the `systemd` tag disables healthchecks silently.
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
           /etc/containers/registries.conf \
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

  # DDEV's own verdict on the provider, which is what a DDEV bug report is about.
  sec "ddev debug dockercheck"
  if ! command -v ddev >/dev/null 2>&1; then
    printf '  ddev is not on PATH; skipped\n'
    DOCKERCHECK_STATUS="ddev not installed"
  elif [ "${PODMAN_SKIP_SMOKE_TEST}" = "true" ]; then
    printf '  skipped (PODMAN_SKIP_SMOKE_TEST=true)\n'
    DOCKERCHECK_STATUS="skipped"
  else
    printf '  (starts containers and runs a trivial build; may take a minute)\n'
    printf '$ ddev debug dockercheck\n'
    local dc_rc esc
    esc=$'\033'
    # Strip colour so the report pastes cleanly into an issue.
    ddev debug dockercheck 2>&1 | sed -e "s/${esc}\[[0-9;]*m//g" -e 's/^/  /'
    dc_rc=${PIPESTATUS[0]}
    printf '  [exit %s]\n' "${dc_rc}"
    [ "${dc_rc}" -eq 0 ] && DOCKERCHECK_STATUS="pass" || DOCKERCHECK_STATUS="FAIL"
  fi

  # The checks are the judging half of the report, and they run the container
  # probes the dump above leaves out.
  sec "checks"
  run_checks || true
  set +o errexit  # run_checks turns it back on as it leaves
  run "systemctl --user list-timers --all --no-pager | tail -5"

  report_summary
  printf '\n===== end of report =====\n'
}

# The dump above is a wall of text. Distil it to the facts a maintainer scans
# first. Judging them is run_checks' job, not this one's.
report_summary() {
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

  local nv nvpath av cli_cg svc_cg linger bus store native ports userns ctx bx
  local ocirt ocipath
  # One podman call for everything. If podman itself is broken its error text is
  # the most useful line in the report, so keep that and skip the derived checks
  # rather than printing a screenful of "?" and false findings.
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
  # Must collapse to one line: a failing docker info emits blank lines, which
  # would wrap the summary and fake a cli/service mismatch.
  svc_cg="$(docker info --format '{{.CgroupDriver}}' 2>/dev/null | tr -d '\r' | grep -v '^$' | tail -1)"
  [ -z "${svc_cg}" ] && svc_cg="?"
  linger="$(loginctl show-user "$(id -un)" --property=Linger 2>/dev/null | cut -d= -f2)"
  [ -S "/run/user/$(id -u)/bus" ] && bus=present || bus=MISSING
  ports="$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null)"
  userns="$(sysctl -n kernel.apparmor_restrict_unprivileged_userns 2>/dev/null || echo 'n/a')"
  ctx="$(docker context show 2>/dev/null)"
  bx="$(docker buildx version 2>/dev/null | awk '{print $2}')"

  sec "summary"
  kv "distro"        "${distro}"
  kv "kernel"        "${kernel}"
  kv "podman"        "${pver}${porigin:+ origin=${porigin}} (${ppkg}) ${ppath}"
  kv "netavark"      "${nv:-?} at ${nvpath:-?} / aardvark ${av:-?}"
  kv "oci runtime"   "${ocirt:-?} ${ocipath}"
  kv "cgroup mgr"    "cli=${cli_cg} service=${svc_cg}"
  kv "linger / bus"  "${linger:-?} / ${bus}"
  kv "storage"       "${store:-?} (native overlay diff: ${native:-?})"
  kv "low ports"     "ip_unprivileged_port_start=${ports:-?}"
  kv "userns"        "apparmor_restrict_unprivileged_userns=${userns}"
  kv "docker ctx"    "${ctx:-?} (buildx ${bx:-none})"
  kv "dockercheck"   "${DOCKERCHECK_STATUS}"
  kv "image load"    "${IMAGELOAD_STATUS}"
  kv "healthchecks"  "${HEALTHCHECK_STATUS}"
  kv "podman -tags"  "$(podman_build_tags || echo 'unknown')"
  kv "policy.json"   "$(active_policy_file 2>/dev/null || echo 'none -- pulls will fail with "no policy.json file found"')"
  # Podman being unusable makes every line above meaningless, so say so here too.
  [ "${info_ok}" -eq 0 ] && kv "podman info" "FAILS: ${info_err}"
  return 0
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
  local action=check
  while [ $# -gt 0 ]; do
    case "$1" in
      --install)       action=install ;;
      --install-brew)  action=install; USE_BREW=true ;;
      --check|-c)      action=check ;;
      --report|-r)     action=report ;;
      --help|-h)       usage; return 0 ;;
      *)
        printf 'Unknown argument: %s\n\n' "$1" >&2
        usage >&2
        exit 2
        ;;
    esac
    shift
  done

  case "${action}" in
    check)   run_checks ;;
    report)  do_report ;;
    install) do_install; run_checks ;;
  esac
}

main "$@"
