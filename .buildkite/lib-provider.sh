#!/usr/bin/env bash
# Shared Docker-provider bring-up/cleanup logic, used by both test.sh
# (correctness tests) and perf.sh (nightly performance benchmarks) so the two
# don't drift out of sync. Extracted verbatim from test.sh's provider handling.
#
# Source, don't execute. Requires DOCKER_TYPE to be set before calling
# provider_bringup. Sets the global TIMEOUT variable as a side effect.

# Find a suitable timeout command for reliability and readability
if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT="gtimeout"
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT="timeout"
else
  echo "Error: Neither 'gtimeout' nor 'timeout' found in PATH." >&2
  exit 1
fi

# Helper: stop orbstack synchronously, bounded by TIMEOUT so a hang can't wedge the job.
function stop_orbstack {
  command -v orb >/dev/null 2>&1 || return 0
  echo "Stopping orbstack"
  ${TIMEOUT} 60s orb stop || true
}

# Helper: start orbstack synchronously, bounded by TIMEOUT, then switch to its docker
# context. orb start is run in the foreground (not backgrounded) so the caller knows
# it has actually run to completion or timed out before moving on; a backgrounded
# orb start can be killed by CI job-cleanup before the VM finishes coming up, leaving
# the docker context set to orbstack but the daemon not running.
function start_orbstack {
  echo "Starting orbstack"
  ${TIMEOUT} 60s orb start || true
  docker context use orbstack || true
}

# Helper: try to remove all containers via an SSH-style shell command.
# Usage: try_cleanup_containers_via_ssh <cmd> [<args>...]
# Returns 0 if containers are clean, 1 if containers remain (deep cleanup needed).
function try_cleanup_containers_via_ssh {
  local -a ssh_cmd=("$@")
  "${ssh_cmd[@]}" bash -lc '
    ids=$(docker ps -aq || true)
    if [ -n "$ids" ]; then
      docker rm -f $ids >/dev/null 2>&1 || true
    fi
    remaining=$(docker ps -aq || true)
    if [ -z "$remaining" ]; then
      echo "No containers remain; skipping docker-state cleanup"
      exit 0
    fi
    echo "CLEANUP REQUIRED: Containers still remain after docker rm -f" >&2
    docker ps -a >&2 || true
    exit 1
  '
}

# Helper: try to remove all containers via the local docker CLI.
# Returns 0 if containers are clean, 1 if containers remain (deep cleanup needed).
function try_cleanup_containers_native {
  local ids
  ids=$(docker ps -aq || true)
  if [ -n "$ids" ]; then
    echo "$ids" | xargs docker rm -f >/dev/null 2>&1 || true
  fi
  local remaining
  remaining=$(docker ps -aq || true)
  if [ -n "$remaining" ]; then
    echo "CLEANUP REQUIRED: Containers still remain after docker rm -f" >&2
    docker ps -a >&2 || true
    return 1
  fi
  echo "No containers remain; skipping docker-state cleanup"
  return 0
}

# provider_bringup brings up whatever Docker provider DOCKER_TYPE names, doing
# whatever provider-specific cleanup is needed first, and blocks until `docker
# ps` succeeds. Call this once per job, after DOCKER_TYPE is set.
#
# Optional: define RAN_TESTS=true and a `cleanup` trap before calling this if
# the caller wants testbot_maintenance.sh to run post-job (see test.sh); this
# function only handles provider startup, not the EXIT trap itself, since
# perf.sh's post-job cleanup needs differ slightly from test.sh's.
function provider_bringup {
  local os
  os=$(go env GOOS)

  # On macOS, we can have several different docker providers, allow testing all
  if [ "${os:-}" = "darwin" ]; then
    # For Lima and Colima, as of Lima 1.0.4, having orbstack running
    # makes lima fail, see https://github.com/lima-vm/lima/issues/3145#issuecomment-2613728408
    stop_orbstack

    case ${DOCKER_TYPE:=none} in
      "colima_vz")
        export COLIMA_INSTANCE=vz
        colima start ${COLIMA_INSTANCE}

        if ! try_cleanup_containers_via_ssh colima ssh -p "${COLIMA_INSTANCE}" --; then
          echo "Performing deep cleanup: removing container state and restarting colima"
          colima ssh -p "${COLIMA_INSTANCE}" -- sudo bash -lc 'rm -rf /var/lib/docker/containers/*'
          colima restart ${COLIMA_INSTANCE}

          local remaining_after_cleanup
          remaining_after_cleanup=$(colima ssh -p "${COLIMA_INSTANCE}" -- bash -lc 'docker ps -aq' || true)
          if [ -n "$remaining_after_cleanup" ]; then
            echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
            colima ssh -p "${COLIMA_INSTANCE}" -- bash -lc 'docker ps -a' >&2 || true
            exit 1
          fi
          echo "Deep cleanup succeeded: all containers removed"
        fi
        docker context use colima-${COLIMA_INSTANCE}
        ;;

      "lima")
        export LIMA_INSTANCE=lima-vz
        export HOMEDIR=/home/testbot.linux
        limactl start ${LIMA_INSTANCE}

        if ! try_cleanup_containers_via_ssh limactl shell "${LIMA_INSTANCE}"; then
          echo "Performing deep cleanup: removing container state and restarting docker"
          limactl shell lima-vz bash -lc "rm -rf ${HOMEDIR}/.local/share/docker/containers/*"
          limactl shell ${LIMA_INSTANCE} systemctl --user restart docker

          for i in {1..30}; do
            if limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps >/dev/null 2>&1'; then
              break
            fi
            echo "Waiting for docker to restart in lima: $i"
            sleep 1
          done

          local remaining_after_cleanup
          remaining_after_cleanup=$(limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps -aq' || true)
          if [ -n "$remaining_after_cleanup" ]; then
            echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
            limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps -a' >&2 || true
            exit 1
          fi
          echo "Deep cleanup succeeded: all containers removed"
        fi
        docker context use lima-${LIMA_INSTANCE}
        ;;

      "docker-desktop")
        open -a Docker
        docker context use desktop-linux
        ;;

      "orbstack")
        start_orbstack
        ;;

      "rancher-desktop")
        ~/.rd/bin/rdctl start
        for i in {1..120}; do
          if docker context use rancher-desktop >/dev/null 2>&1 ; then
            break
          fi
          echo "$(date): Waiting for rancher-desktop context to be available"
          sleep 1
        done
        docker context use rancher-desktop
        ;;

      "podman-rootless")
        podman machine start
        docker context use podman-rootless

        if ! try_cleanup_containers_via_ssh podman machine ssh; then
          echo "ERROR: Containers remain after podman machine start; aborting" >&2
          docker ps -a >&2 || true
          exit 1
        fi
        podman machine ssh 'sudo fstrim -av'
        ;;

      *)
        echo "no DOCKER_TYPE specified, exiting" && exit 10
        ;;
    esac
  fi

  # Handle docker-ce cleanup for WSL and other native Docker CE instances
  if [ "${DOCKER_TYPE:-}" = "docker-ce" ] || [ "${DOCKER_TYPE:-}" = "wsl2dockerinside" ]; then
    if ! try_cleanup_containers_native; then
      echo "Performing deep cleanup: removing container state and restarting docker"
      sudo bash -c "rm -rf /var/lib/docker/containers/*"
      sudo systemctl restart docker

      for i in {1..30}; do
        if docker ps >/dev/null 2>&1 ; then
          break
        fi
        echo "Waiting for docker to restart: $i"
        sleep 1
      done

      local remaining_after_cleanup
      remaining_after_cleanup=$(docker ps -aq || true)
      if [ -n "$remaining_after_cleanup" ]; then
        echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
        docker ps -a >&2 || true
        exit 1
      fi
      echo "Deep cleanup succeeded: all containers removed"
    fi
  fi

  # On Windows, start Docker Desktop if this job uses it, before waiting on docker.
  # This is the Windows analogue of the lima/colima/orbstack provider startup above;
  # it is provider setup, separate from any test/perf-specific sanity checks. No-op on
  # non-Windows and on docker-ce-inside cases.
  bash "$(dirname "${BASH_SOURCE[0]}")/start-docker-desktop.sh" || true

  # Make sure docker is working
  echo "Waiting for docker provider to come up: $(date)"
  date && ${TIMEOUT} 3m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
    sleep 10
    echo "Waiting: $(date)"
  done'
  echo "Testing again to make sure docker came up: $(date)"
  if ! docker ps >/dev/null 2>&1 ; then
    echo "Docker is not running, exiting"
    exit 1
  fi
}

# provider_shutdown stops whatever Docker provider is running, leaving either
# Orbstack or Docker Desktop running afterward (matches test.sh's long-standing
# cleanup trap behavior, extracted here so perf.sh can reuse it verbatim).
function provider_shutdown {
  stop_orbstack
  if [ -f /Applications/Docker.app ]; then echo "Stopping Docker Desktop" && (killall com.docker.backend || true); fi
  command -v colima 2>/dev/null && echo "Stopping colima_vz" && (colima stop -f vz || true)
  command -v limactl 2>/dev/null && echo "Stopping lima" && (limactl stop -f lima-vz || true)
  if [ -f ~/.rd/bin/rdctl ]; then echo "Stopping Rancher Desktop" && (~/.rd/bin/rdctl shutdown || true); fi
  command -v podman 2>/dev/null && echo "Stopping podman machine" && (podman machine stop || true)
  docker context use default || true
  # Leave orbstack running as the most likely to be reliable, otherwise Docker Desktop.
  if command -v orb 2>/dev/null ; then
    start_orbstack
  else
    docker context use desktop-linux || true
    open -a Docker || true
    sleep 5
  fi
}
