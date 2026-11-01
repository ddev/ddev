#!/usr/bin/env bash
# Scheduled disk cleanup of a macOS testbot, run by testbot-disk-cleanup.yml.
# Per-job testbot_maintenance.sh only cleans the provider that job used and
# keeps old images cached; this starts every provider on the machine in turn,
# removes all its containers, images, volumes and build cache, returns the
# freed blocks to the host, and recreates a VM whose disk is still too full.
# CLEANUP_PROVIDERS overrides the detected list, e.g. "podman-rootless lima".

set -eu -o pipefail

# shellcheck source=lib-provider.sh
source "$(dirname "$0")/lib-provider.sh"
# shellcheck source=lib-disk.sh
source "$(dirname "$0")/lib-disk.sh"

# A VM whose docker disk is fuller than this after pruning gets recreated.
VM_RESET_PERCENT=${VM_RESET_PERCENT:-70}

if [ "$(go env GOOS)" != "darwin" ]; then
  echo "testbot_disk_cleanup.sh only handles macOS testbots" && exit 1
fi

providers=${CLEANUP_PROVIDERS:-}
if [ -z "${providers}" ]; then
  if command -v colima >/dev/null && [ -d ~/.colima/vz ]; then providers+=" colima_vz"; fi
  if command -v limactl >/dev/null && [ -d ~/.lima/lima-vz ]; then providers+=" lima"; fi
  if [ -x ~/.rd/bin/rdctl ]; then providers+=" rancher-desktop"; fi
  if command -v podman >/dev/null && podman machine inspect >/dev/null 2>&1; then providers+=" podman-rootless"; fi
  if command -v orb >/dev/null; then providers+=" orbstack"; fi
fi

function provider_stop {
  case "$1" in
    colima_vz) colima stop -f vz ;;
    lima) limactl stop -f lima-vz ;;
    rancher-desktop) ~/.rd/bin/rdctl shutdown ;;
    podman-rootless) podman machine stop ;;
    orbstack) stop_orbstack ;;
  esac
}

# docker_disk_percent prints how full the filesystem under the provider's
# container storage is, as seen from a container's root.
function docker_disk_percent {
  docker run --rm ddev/ddev-utilities df -P / | awk 'NR==2 {gsub(/%/, ""); print $5}'
}

# provider_reset recreates the VM, keeping its CPU, memory and disk settings.
# Only providers whose recreation needs no manual setup are handled.
function provider_reset {
  case "$1" in
    podman-rootless)
      local cpus memory disk
      read -r cpus memory disk < <(podman machine inspect --format '{{.Resources.CPUs}} {{.Resources.Memory}} {{.Resources.DiskSize}}')
      podman machine rm -f
      podman machine init --cpus "${cpus}" --memory "${memory}" --disk-size "${disk}"
      ;;
    lima)
      limactl stop -f lima-vz || true
      limactl factory-reset lima-vz
      ;;
    *)
      echo "WARNING: $1 is still over ${VM_RESET_PERCENT}% full and must be recreated by hand"
      return 1
      ;;
  esac
}

function cleanup_provider {
  DOCKER_TYPE=$1
  provider_bringup
  docker rm -f $(docker ps -aq) >/dev/null 2>&1 || true
  docker system prune -a --volumes --force || true
  docker volume prune -a -f || true
  # Podman's Docker API has no build cache endpoint.
  if [ "${DOCKER_TYPE}" != "podman-rootless" ]; then
    docker builder prune -a -f || true
  fi
  provider_fstrim "${DOCKER_TYPE}" || true

  local used
  used=$(docker_disk_percent || true)
  echo "${DOCKER_TYPE}: container storage ${used:-unknown}% full after pruning"
  provider_stop "${DOCKER_TYPE}" || true
  if [ -n "${used}" ] && [ "${used}" -gt "${VM_RESET_PERCENT}" ]; then
    echo "Recreating ${DOCKER_TYPE}'s VM"
    provider_reset "${DOCKER_TYPE}"
  fi
}

# Apple's container CLI manages its own storage, with or without socktainer.
function cleanup_apple_container {
  container system start --enable-kernel-install
  container prune || true
  container image prune --all || true
  container volume prune || true
  container system stop || true
}

trap provider_shutdown EXIT

echo "--- $(hostname): $(free_gb)GB free before cleanup; providers:${providers}"
failed=""
for p in ${providers}; do
  echo "--- cleaning ${p}"
  set +e
  (set -e; cleanup_provider "${p}")
  rc=$?
  set -e
  if [ "${rc}" -ne 0 ]; then
    failed+=" ${p}"
    provider_stop "${p}" || true
  fi
done

if command -v container >/dev/null; then
  echo "--- cleaning apple container"
  cleanup_apple_container || failed+=" apple-container"
fi

host_disk_cleanup
echo "--- $(hostname): $(free_gb)GB free after cleanup"
if [ -n "${failed}" ]; then
  echo "Cleanup failed for:${failed}" && exit 1
fi
