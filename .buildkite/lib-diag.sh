#!/usr/bin/env bash
# Failure reporting for the Buildkite startup scripts (test.sh, perf.sh,
# lib-provider.sh, sanetestbot.sh), so a failed or timed-out startup says
# which step failed, how, and what the Docker provider looked like at the
# time, instead of a bare "exit status 124". Source, don't execute.

if [ -z "${TIMEOUT:-}" ]; then
  if command -v gtimeout >/dev/null 2>&1; then
    TIMEOUT="gtimeout"
  elif command -v timeout >/dev/null 2>&1; then
    TIMEOUT="timeout"
  fi
fi

# bounded <limit> <command> [args...] runs an executable (not a shell
# function) for at most <limit>, such as 30s or 5m, and exits 124 when the
# limit is hit. A command that ignores SIGTERM is killed 10s later.
function bounded {
  local limit=$1
  shift
  if [ -n "${TIMEOUT:-}" ]; then
    ${TIMEOUT} -k 10s "${limit}" "$@"
  else
    "$@"
  fi
}

# describe_rc <rc> <limit> prints how a bounded command ended.
function describe_rc {
  case "$1" in
    124) echo "timed out after $2" ;;
    137) echo "was killed after ignoring SIGTERM at its $2 limit" ;;
    *) echo "exited with status $1" ;;
  esac
}

# quote_cmd prints a command the way a shell would read it back.
function quote_cmd {
  printf '%q ' "$@" | sed 's/ $//'
}

# diag_cmd <limit> <command> [args...] prints a command and its output for a
# diagnostics report. It never fails, and a hung daemon can only stall it
# for <limit>.
function diag_cmd {
  local limit=$1 rc=0
  shift
  echo "\$ $(quote_cmd "$@")"
  bounded "${limit}" "$@" </dev/null 2>&1 || rc=$?
  if [ "${rc}" != 0 ]; then
    echo "($(describe_rc "${rc}" "${limit}"))"
  fi
}

# macos_privacy_prompts reports macOS privacy (TCC) prompts from the last 15
# minutes. An unanswered prompt, such as Rancher Desktop asking to access
# files on a network volume, blocks the VM's file sharing until someone
# clicks Allow on the testbot, so docker commands that touch host files hang.
function macos_privacy_prompts {
  [ "$(uname -s)" = "Darwin" ] || return 0
  local prompts
  prompts=$(bounded 30s log show --last 15m --style compact \
    --predicate 'subsystem == "com.apple.TCC" AND eventMessage CONTAINS "AUTHREQ_PROMPTING"' 2>/dev/null |
    sed -nE 's/.*service=([^,]+), subject=Sub:\{([^}]+)\}.*/\2 asked for \1/p' | sort -u || true)
  if [ -n "${prompts}" ]; then
    echo "macOS privacy prompts on $(hostname) in the last 15 minutes:"
    echo "${prompts}"
    echo "If nobody answered a prompt, it is still waiting. Click Allow on the testbot's screen,"
    echo "or give the app Full Disk Access in System Settings > Privacy & Security."
  fi
}

# docker_diagnostics prints the state of the Docker provider and the host.
# The privacy prompt check comes first, so it shows in the expanded section.
function docker_diagnostics {
  macos_privacy_prompts
  echo "--- Diagnostics at $(date) on $(hostname): DOCKER_TYPE=${DOCKER_TYPE:-notset} DOCKER_CONTEXT=${DOCKER_CONTEXT:-notset}"
  diag_cmd 20s docker context ls
  diag_cmd 20s docker version
  diag_cmd 20s docker ps -a
  diag_cmd 30s docker info
  # The provider's own view, which can show a VM that is up behind a docker
  # socket that no longer answers.
  case "${DOCKER_TYPE:-}" in
    colima_vz) diag_cmd 20s colima status -p vz ;;
    lima) diag_cmd 20s limactl list ;;
    orbstack) diag_cmd 20s orb status ;;
    rancher-desktop)
      diag_cmd 20s "${HOME}/.rd/bin/rdctl" version
      diag_cmd 30s "${HOME}/.rd/bin/rdctl" shell sh -c 'uptime; docker ps -a'
      ;;
    podman-rootless) diag_cmd 20s podman machine list ;;
    socktainer) diag_cmd 20s container system status ;;
    docker-desktop | dockerforwindows | wsl2-docker-desktop) diag_cmd 20s docker desktop status ;;
  esac
  # A port already held by another provider makes `docker run -p 80:80` hang.
  if command -v lsof >/dev/null 2>&1; then
    diag_cmd 20s lsof -nP -iTCP:80 -iTCP:443 -sTCP:LISTEN
  fi
  diag_cmd 10s uptime
  diag_cmd 10s df -h "${HOME}"
}

# report_failure <description> <limit> <rc> [command...] opens an expanded
# Buildkite section naming what failed and the command that ran.
function report_failure {
  local what=$1 limit=$2 rc=$3
  shift 3
  echo "+++ FAILED: ${what} $(describe_rc "${rc}" "${limit}")"
  if [ $# -gt 0 ]; then
    echo "Command: $(quote_cmd "$@")"
  fi
}

# run_step <name> <limit> <command> [args...] runs a short check quietly. It
# prints one timestamped line when it passes; when it fails or times out it
# prints its output, as far as it got, and the diagnostics, and returns its
# exit status.
function run_step {
  local name=$1 limit=$2 rc=0 out start=${SECONDS}
  shift 2
  CURRENT_STEP=${name}
  out=$(bounded "${limit}" "$@" 2>&1) || rc=$?
  if [ "${rc}" = 0 ]; then
    echo "$(date '+%H:%M:%S') ok: ${name} ($((SECONDS - start))s)"
    return 0
  fi
  report_failure "${name}" "${limit}" "${rc}" "$@"
  echo "Output:"
  echo "${out:-(no output)}"
  docker_diagnostics
  return "${rc}"
}

# run_phase <name> <limit> <command> [args...] runs a long startup phase with
# its output streamed, and reports a failure or timeout the way run_step does.
function run_phase {
  local name=$1 limit=$2 rc=0 start=${SECONDS}
  shift 2
  CURRENT_PHASE=${name}
  bounded "${limit}" "$@" || rc=$?
  if [ "${rc}" = 0 ]; then
    echo "$(date '+%H:%M:%S') ${name} finished in $((SECONDS - start))s"
    return 0
  fi
  report_failure "${name}" "${limit}" "${rc}" "$@"
  docker_diagnostics
  return "${rc}"
}
