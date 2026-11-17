#!/usr/bin/env bash
# Shared Docker-provider bring-up/cleanup logic, used by both test.sh
# (correctness tests) and perf.sh (nightly performance benchmarks) so the two
# don't drift out of sync. Extracted verbatim from test.sh's provider handling.
#
# Source, don't execute. Requires DOCKER_TYPE to be set before calling
# provider_bringup. Sets the global TIMEOUT variable as a side effect.
#
# On macOS, provider_bringup selects the provider by exporting DOCKER_CONTEXT
# rather than running `docker context use`, so several providers can stay
# running side by side and each job talks only to its own. A provider that
# switches the global context when it starts (colima, orbstack) can't redirect
# a job, because DOCKER_CONTEXT overrides the global setting. Lima and colima
# jobs stop orbstack first.
#
# PROVIDER_EXCLUSIVE=true restores the old behavior of stopping every other
# provider first; perf.sh uses it so idle VMs don't skew benchmark results.
PROVIDER_EXCLUSIVE=${PROVIDER_EXCLUSIVE:-false}

# Docker contexts of the macOS providers that can share a testbot.
MACOS_PROVIDER_CONTEXTS="colima-vz lima-lima-vz desktop-linux orbstack rancher-desktop podman-rootless socktainer"

# Find a suitable timeout command for reliability and readability
if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT="gtimeout"
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT="timeout"
else
  echo "Error: Neither 'gtimeout' nor 'timeout' found in PATH." >&2
  exit 1
fi

# shellcheck source=lib-diag.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib-diag.sh"

# Helper: stop orbstack synchronously, bounded by TIMEOUT so a hang can't wedge the job.
function stop_orbstack {
  command -v orb >/dev/null 2>&1 || return 0
  echo "Stopping orbstack"
  ${TIMEOUT} 60s orb stop || true
}

# Helper: start orbstack synchronously, bounded by TIMEOUT. orb start is run in the
# foreground (not backgrounded) so the caller knows it has actually run to completion
# or timed out before moving on; a backgrounded orb start can be killed by CI
# job-cleanup before the VM finishes coming up.
function start_orbstack {
  echo "Starting orbstack"
  ${TIMEOUT} 60s orb start || true
}

# Helper: succeed if the daemon behind docker context $1 answers.
function context_is_up {
  ${TIMEOUT} 20s docker --context "$1" ps >/dev/null 2>&1
}

# Helper: stop the provider named by a DOCKER_TYPE value.
function stop_provider {
  case "$1" in
    "orbstack")
      stop_orbstack
      ;;
    "docker-desktop")
      # Stopping Docker Desktop is unreliable, so leave it running.
      echo "Not stopping Docker Desktop"
      ;;
    "colima_vz")
      command -v colima >/dev/null 2>&1 && echo "Stopping colima_vz" && (${TIMEOUT} 120s colima stop -f vz || true)
      ;;
    "lima")
      command -v limactl >/dev/null 2>&1 && echo "Stopping lima" && (${TIMEOUT} 120s limactl stop -f lima-vz || true)
      ;;
    "rancher-desktop")
      if [ -f ~/.rd/bin/rdctl ]; then echo "Stopping Rancher Desktop" && (${TIMEOUT} 120s ~/.rd/bin/rdctl shutdown || true); fi
      ;;
    "podman-rootless")
      command -v podman >/dev/null 2>&1 && echo "Stopping podman machine" && (${TIMEOUT} 120s podman machine stop || true)
      ;;
    "socktainer")
      if command -v container >/dev/null 2>&1; then
        echo "Stopping socktainer and apple container"
        pkill -x socktainer || true
        ${TIMEOUT} 60s container system stop || true
      fi
      ;;
  esac
  return 0
}

# Helper: print the DOCKER_TYPE of the provider that runs the command line $1,
# or nothing for a process that belongs to no provider. Rancher Desktop and
# colima run their own copies of Lima's limactl, so check them before Lima.
function provider_of_command {
  case "$1" in
    *"Rancher Desktop"* | *rancher-desktop*) echo "rancher-desktop" ;;
    *limactl*colima* | *colima*limactl*) echo "colima_vz" ;;
    *limactl*lima-vz* | *lima-vz*limactl*) echo "lima" ;;
    *OrbStack*) echo "orbstack" ;;
    *com.docker.* | *Docker.app*) echo "docker-desktop" ;;
    *gvproxy* | *podman* | *vfkit* | *krunkit*) echo "podman-rootless" ;;
    *socktainer* | *com.apple.container*) echo "socktainer" ;;
  esac
}

# Helper: print "<provider> <pid> <port>" for each TCP port that a provider
# process listens on.
function provider_listeners {
  local pid="" line cmd provider
  lsof -nP -iTCP -sTCP:LISTEN -Fpn 2>/dev/null | while read -r line; do
    case "${line}" in
      p*) pid=${line#p} ;;
      n*)
        cmd=$(ps -o command= -p "${pid}" 2>/dev/null || true)
        provider=$(provider_of_command "${cmd}")
        [ -n "${provider}" ] && echo "${provider} ${pid} ${line##*:}"
        ;;
    esac
  done | sort -u
}

# Helper: print the host ports this job's router will use.
function own_router_ports {
  case "${DOCKER_TYPE}" in
    "podman-rootless" | "socktainer") echo "8080 8443 8025 8026 8142 8143 10999" ;;
    *) echo "80 443 8025 8026 8142 8143 10999" ;;
  esac
}

# Helper: print the problems that would make tests run on substitute ports:
# another provider listening on any port, or a busy router port of this job.
# Docker Desktop counts only through the router ports, because it isn't stopped.
function provider_port_problems {
  local port
  provider_listeners | awk -v own="${DOCKER_TYPE}" '$1 != own && $1 != "docker-desktop" { print "other " $1 " pid " $2 " listens on " $3 }'
  for port in $(own_router_ports); do
    if nc -z -G 2 127.0.0.1 "${port}" >/dev/null 2>&1; then
      echo "busy ${port} held by: $(lsof -nP -t -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null | while read -r p; do ps -o command= -p "${p}"; done | tr '\n' ' ')"
    fi
  done
}

# Helper: wait up to $1 seconds for provider_port_problems to report nothing.
function wait_provider_ports_clean {
  local i problems
  for ((i = 0; i < $1; i++)); do
    problems=$(provider_port_problems)
    [ -z "${problems}" ] && return 0
    sleep 1
  done
  echo "${problems}"
  return 1
}

# Helper: make sure no provider holds ports that this job's router or tests
# may need. Every container is gone by now, so a port that another provider
# still listens on is a stale forward, and stopping that provider loses
# nothing. Returns 1 when this job's own provider holds one of its router
# ports and needs a restart. Fails the job when a port stays busy.
function ensure_provider_ports_clean {
  local problems others restart_own=false provider port pid
  # A forward can take a few seconds to close after its container is removed.
  problems=$(wait_provider_ports_clean 15) && return 0
  echo "Docker provider port problems:"
  echo "${problems}"
  others=$(echo "${problems}" | awk '$1 == "other" { print $2 }' | sort -u)
  for provider in ${others}; do
    stop_provider "${provider}"
  done
  for port in $(echo "${problems}" | awk '$1 == "busy" { print $2 }'); do
    for pid in $(lsof -nP -t -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null); do
      [ "$(provider_of_command "$(ps -o command= -p "${pid}")")" = "${DOCKER_TYPE}" ] && restart_own=true
    done
  done
  if [ "${restart_own}" = "true" ]; then
    if [ "${PROVIDER_PORTS_RESTARTED:-false}" = "true" ] || [ "${DOCKER_TYPE}" = "docker-desktop" ]; then
      echo "ERROR: ${DOCKER_TYPE} holds its router ports, and a restart did not or cannot free them" >&2
      lsof -nP -iTCP -sTCP:LISTEN >&2 || true
      exit 1
    fi
    return 1
  fi
  problems=$(wait_provider_ports_clean 30) && return 0
  echo "ERROR: ports are still busy before the tests start:" >&2
  echo "${problems}" >&2
  lsof -nP -iTCP -sTCP:LISTEN >&2 || true
  exit 1
}

# Helper: remove all containers in every other provider that is running, so a
# container left behind by an earlier job (a ddev-router holding ports 80/443,
# for instance) can't block this one. Assumes one job at a time per testbot.
function cleanup_other_provider_containers {
  local ctx ids
  for ctx in ${MACOS_PROVIDER_CONTEXTS}; do
    [ "${ctx}" = "${DOCKER_CONTEXT}" ] && continue
    docker context inspect "${ctx}" >/dev/null 2>&1 || continue
    ids=$(${TIMEOUT} 20s docker --context "${ctx}" ps -aq 2>/dev/null || true)
    if [ -n "${ids}" ]; then
      echo "Removing $(echo "${ids}" | wc -l | tr -d ' ') leftover container(s) in ${ctx}"
      echo "${ids}" | xargs ${TIMEOUT} 60s docker --context "${ctx}" rm -f >/dev/null 2>&1 || true
    fi
  done
  # Apple container runs each container as its own launchd service, so its
  # containers can outlive socktainer, where the docker API can't reach them.
  if [ "${DOCKER_TYPE}" != "socktainer" ] && command -v container >/dev/null 2>&1; then
    echo "Stopping socktainer and apple container"
    pkill -x socktainer || true
    ${TIMEOUT} 60s container system stop || true
  fi
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
    # With orbstack running, a lima or colima container that publishes ports
    # 80/443 hangs, which times out sanetestbot.sh.
    if [ "${PROVIDER_EXCLUSIVE}" = "true" ] || [ "${DOCKER_TYPE:-}" = "lima" ] || [ "${DOCKER_TYPE:-}" = "colima_vz" ]; then
      stop_orbstack
    fi

    case ${DOCKER_TYPE:=none} in
      "colima_vz")
        export COLIMA_INSTANCE=vz
        export DOCKER_CONTEXT=colima-${COLIMA_INSTANCE}
        context_is_up "${DOCKER_CONTEXT}" || colima start ${COLIMA_INSTANCE}

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
        ;;

      "lima")
        export LIMA_INSTANCE=lima-vz
        export HOMEDIR=/home/testbot.linux
        export DOCKER_CONTEXT=lima-${LIMA_INSTANCE}
        context_is_up "${DOCKER_CONTEXT}" || limactl start ${LIMA_INSTANCE}

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
        ;;

      "docker-desktop")
        export DOCKER_CONTEXT=desktop-linux
        open -a Docker
        ;;

      "orbstack")
        export DOCKER_CONTEXT=orbstack
        context_is_up "${DOCKER_CONTEXT}" || start_orbstack
        ;;

      "rancher-desktop")
        export DOCKER_CONTEXT=rancher-desktop
        if ! context_is_up "${DOCKER_CONTEXT}"; then
          echo "$(date): rancher-desktop is not answering, running rdctl start"
          ~/.rd/bin/rdctl start
        fi
        for i in {1..120}; do
          if docker context inspect rancher-desktop >/dev/null 2>&1 ; then
            break
          fi
          if [ "$((i % 10))" = 1 ]; then
            echo "$(date): Waiting for rancher-desktop context to be available ($i/120)"
          fi
          sleep 1
        done
        ;;

      "podman-rootless")
        export DOCKER_CONTEXT=podman-rootless
        if [ "$(podman machine inspect --format '{{.State}}')" != "running" ]; then
          podman machine start
        fi
        # The API socket lives under $TMPDIR, which differs between the agent
        # and an ssh session, so follow wherever this start put it.
        docker context update podman-rootless --docker "host=unix://$(podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}')" >/dev/null

        # podman machine ssh joins its arguments into one string, which breaks
        # the bash -lc script in try_cleanup_containers_via_ssh.
        if ! try_cleanup_containers_native; then
          echo "ERROR: Containers remain after podman machine start; aborting" >&2
          docker ps -a >&2 || true
          exit 1
        fi
        podman machine ssh 'sudo fstrim -av'
        ;;

      "socktainer")
        export DOCKER_CONTEXT=socktainer
        if ! context_is_up "${DOCKER_CONTEXT}"; then
          # Neither runs as a brew service on the testbots, so this job owns both.
          container system start --disable-kernel-install
          nohup socktainer --no-auto-start >~/tmp/socktainer.log 2>&1 &
          for i in {1..60}; do
            if docker --context socktainer ps >/dev/null 2>&1; then
              break
            fi
            echo "Waiting for socktainer: $i"
            sleep 1
          done
        fi

        if ! try_cleanup_containers_native; then
          echo "ERROR: Containers remain after socktainer start; aborting" >&2
          exit 1
        fi
        ;;

      *)
        echo "no DOCKER_TYPE specified, exiting" && exit 10
        ;;
    esac
    echo "Using DOCKER_CONTEXT=${DOCKER_CONTEXT}"

    if [ "${PROVIDER_EXCLUSIVE}" != "true" ]; then
      cleanup_other_provider_containers
    fi
    if ! ensure_provider_ports_clean; then
      echo "Restarting ${DOCKER_TYPE} to drop a stale port forward"
      stop_provider "${DOCKER_TYPE}"
      PROVIDER_PORTS_RESTARTED=true
      provider_bringup
      return
    fi
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

  # Make sure docker is working. Each docker ps is bounded too, because one
  # that hangs would otherwise use up the whole wait without saying so. The
  # `|| rc=$?` keeps set -e from exiting silently when the wait times out.
  echo "Waiting for docker provider ${DOCKER_CONTEXT:-${DOCKER_TYPE:-}} to come up: $(date)"
  local rc=0
  ${TIMEOUT} 3m bash -c "while ! ${TIMEOUT} 30s docker ps >/dev/null 2>&1 ; do
    echo \"Waiting for docker ps: \$(date)\"
    sleep 10
  done" || rc=$?
  if [ "${rc}" != 0 ]; then
    report_failure "waiting for the docker provider to answer docker ps" 3m "${rc}"
    docker_diagnostics
    exit 1
  fi
  echo "Docker provider is up: $(date)"
}

# provider_shutdown stops whatever Docker provider is running, leaving either
# Orbstack or Docker Desktop running afterward. Used by PROVIDER_EXCLUSIVE
# callers (perf.sh) and testbot_disk_cleanup.sh; test.sh leaves providers up.
function provider_shutdown {
  local provider
  # Apple container runs each container as its own launchd service, so a
  # socktainer job killed before its cleanup leaves a ddev-router holding host
  # ports. Stop it first, because a canceled job may not get further.
  for provider in socktainer orbstack docker-desktop colima_vz lima rancher-desktop podman-rootless; do
    stop_provider "${provider}"
  done
  restore_resting_provider
}

function restore_resting_provider {
  if command -v orb >/dev/null 2>&1 ; then
    context_is_up orbstack || start_orbstack
    docker context use orbstack >/dev/null || true
  else
    docker context use desktop-linux >/dev/null || true
    open -a Docker || true
    sleep 5
  fi
}
