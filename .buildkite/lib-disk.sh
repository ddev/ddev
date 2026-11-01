#!/usr/bin/env bash
# Disk-space helpers shared by testbot_maintenance.sh (every job) and
# testbot_disk_cleanup.sh (the scheduled sweep of every provider on a macOS
# testbot). Source, don't execute.

# Below this many GB free on the host, testbot_maintenance.sh prunes all
# unused images and the Go build cache instead of only dangling images.
LOW_DISK_GB=${LOW_DISK_GB:-40}

# free_gb prints the GB available on the filesystem holding $HOME.
function free_gb {
  df -k "$HOME" | awk 'NR==2 {print int($4/1048576)}'
}

# provider_fstrim returns blocks freed inside a provider's VM to the host;
# without it the VM's sparse disk file stays at its high-water mark.
# OrbStack and Docker Desktop trim on their own.
function provider_fstrim {
  case "${1:-}" in
    colima_vz) colima ssh -p vz -- sudo fstrim -av ;;
    lima) limactl shell lima-vz sudo fstrim -av ;;
    # Rancher Desktop's Alpine VM has BusyBox fstrim, which has no -a.
    rancher-desktop) ~/.rd/bin/rdctl shell sudo sh -c 'for m in $(awk "\$3 ~ /^(ext4|xfs|btrfs)\$/ {print \$2}" /proc/mounts); do fstrim -v "$m"; done' ;;
    podman-rootless) podman machine ssh 'sudo fstrim -av' ;;
  esac
}

# host_disk_cleanup removes host-side leftovers that no test needs.
function host_disk_cleanup {
  # Mutagen keeps a cache per sync session; with no sessions left, they are orphans.
  local mutagen_dir=~/.ddev_mutagen_data_directory
  if [ -d "${mutagen_dir}/caches" ] && [ -z "$(ls -A "${mutagen_dir}/sessions" 2>/dev/null)" ]; then
    echo "Removing orphaned Mutagen caches in ${mutagen_dir}/caches"
    rm -rf "${mutagen_dir}/caches"
  fi
  # buildx writes a ref file per build and never removes them.
  if [ -d ~/.docker/buildx/refs ]; then
    find ~/.docker/buildx/refs -type f -mtime +7 -delete 2>/dev/null || true
  fi
}
