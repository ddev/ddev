#!/bin/bash
# Runs health checks on a macOS testbot and pushes each result to Icinga as a
# passive check result. launchd runs it every 5 minutes; the setup is in
# "Icinga2 Monitoring for macOS and Windows Runners" in buildkite-testmachine-setup.md.
#
# Reads ICINGA_URL, ICINGA_HOST, ICINGA_API_USER and ICINGA_API_PASSWORD from
# ~/.config/ddev-icinga/env, and the Icinga CA from ~/.config/ddev-icinga/ca.crt.
# With --dry-run it prints the results instead of sending them.

set -u -o pipefail
export PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin

CONFIG_DIR=${DDEV_ICINGA_CONFIG_DIR:-${HOME}/.config/ddev-icinga}
DRY_RUN=false
if [ "${1:-}" = "--dry-run" ]; then DRY_RUN=true; fi

DATA_VOLUME=/System/Volumes/Data
DISK_WARN_PCT=${DISK_WARN_PCT:-80}
DISK_CRIT_PCT=${DISK_CRIT_PCT:-90}
LOAD_WARN=${LOAD_WARN:-7}
LOAD_CRIT=${LOAD_CRIT:-10}
JOB_WARN_HOURS=${JOB_WARN_HOURS:-4}
JOB_CRIT_HOURS=${JOB_CRIT_HOURS:-8}
PROBE_TIMEOUT=${PROBE_TIMEOUT:-20s}

OK=0
WARNING=1
CRITICAL=2
UNKNOWN=3

if [ -f "${CONFIG_DIR}/env" ]; then
  # shellcheck source=/dev/null
  . "${CONFIG_DIR}/env"
fi
ICINGA_HOST=${ICINGA_HOST:-$(hostname -s)}

TIMEOUT=$(command -v gtimeout || command -v timeout || true)

# push <service> <exit_status> <output> [<perfdata>...]
# An empty service name sends the host check result.
function push {
  local service=$1 status=$2 output=$3
  shift 3
  local perfdata body
  perfdata=$(printf '%s\n' "$@" | jq -R . | jq -sc 'map(select(length > 0))')
  if [ -z "${service}" ]; then
    body=$(jq -nc --arg h "${ICINGA_HOST}" --argjson s "${status}" --arg o "${output}" \
      --argjson p "${perfdata}" --arg src "$(hostname -s)" \
      '{type: "Host", filter: "host.name == h", filter_vars: {h: $h},
        exit_status: $s, plugin_output: $o, performance_data: $p, check_source: $src}')
  else
    body=$(jq -nc --arg h "${ICINGA_HOST}" --arg n "${service}" --argjson s "${status}" \
      --arg o "${output}" --argjson p "${perfdata}" --arg src "$(hostname -s)" \
      '{type: "Service", filter: "host.name == h && service.name == n",
        filter_vars: {h: $h, n: $n},
        exit_status: $s, plugin_output: $o, performance_data: $p, check_source: $src}')
  fi

  if [ "${DRY_RUN}" = "true" ]; then
    echo "${service:-<host>} [${status}] ${output} | $*"
    return 0
  fi
  if ! curl -sS --fail-with-body --max-time 30 \
    --cacert "${CONFIG_DIR}/ca.crt" \
    -u "${ICINGA_API_USER}:${ICINGA_API_PASSWORD}" \
    -H 'Accept: application/json' \
    -X POST "${ICINGA_URL}/v1/actions/process-check-result" \
    -d "${body}" >/dev/null; then
    echo "Failed to push ${service:-host} result to ${ICINGA_URL}" >&2
  fi
}

# etime_seconds converts ps etime ([[dd-]hh:]mm:ss) to seconds.
function etime_seconds {
  awk -v t="$1" 'BEGIN {
    d = 0; if (index(t, "-")) { split(t, a, "-"); d = a[1]; t = a[2] }
    n = split(t, p, ":"); s = 0
    for (i = 1; i <= n; i++) s = s * 60 + p[i]
    print d * 86400 + s
  }'
}

function check_host {
  push "" "${OK}" "UP: $(uptime | sed 's/^ *//')"
}

function check_disk {
  local line used_pct free_gb status=${OK}
  line=$(df -k "${DATA_VOLUME}" | awk 'NR==2')
  used_pct=$(echo "${line}" | awk '{ printf "%d", $3 * 100 / ($3 + $4) }')
  free_gb=$(echo "${line}" | awk '{ printf "%d", $4 / 1048576 }')
  if [ "${used_pct}" -ge "${DISK_CRIT_PCT}" ]; then
    status=${CRITICAL}
  elif [ "${used_pct}" -ge "${DISK_WARN_PCT}" ]; then
    status=${WARNING}
  fi
  push "disk" "${status}" "${DATA_VOLUME} ${used_pct}% used, ${free_gb}GB free" \
    "used_pct=${used_pct}%;${DISK_WARN_PCT};${DISK_CRIT_PCT};0;100" "free=${free_gb}GB"
}

function check_load {
  local load1 load5 load15 status=${OK}
  read -r load1 load5 load15 < <(sysctl -n vm.loadavg | tr -d '{}')
  if awk -v l="${load15}" -v c="${LOAD_CRIT}" 'BEGIN { exit !(l >= c) }'; then
    status=${CRITICAL}
  elif awk -v l="${load15}" -v w="${LOAD_WARN}" 'BEGIN { exit !(l >= w) }'; then
    status=${WARNING}
  fi
  push "load-average" "${status}" "load average ${load1}, ${load5}, ${load15}" \
    "load1=${load1}" "load5=${load5}" "load15=${load15};${LOAD_WARN};${LOAD_CRIT}"
}

# check_buildkite_agent wants the agent running and no job older than the limits.
function check_buildkite_agent {
  local status=${OK} msg longest=0 secs etime hours
  if ! pgrep -qx buildkite-agent; then
    push "buildkite-agent" "${CRITICAL}" "buildkite-agent is not running"
    return
  fi
  for etime in $(ps -axo etime=,command= | awk '/buildkite-agent bootstrap/ && !/awk/ { print $1 }'); do
    secs=$(etime_seconds "${etime}")
    [ "${secs}" -gt "${longest}" ] && longest=${secs}
  done
  hours=$((longest / 3600))
  if [ "${longest}" -eq 0 ]; then
    msg="buildkite-agent running, idle"
  else
    msg="buildkite-agent running, current job ${hours}h $(((longest % 3600) / 60))m old"
  fi
  if [ "${hours}" -ge "${JOB_CRIT_HOURS}" ]; then
    status=${CRITICAL}
  elif [ "${hours}" -ge "${JOB_WARN_HOURS}" ]; then
    status=${WARNING}
  fi
  push "buildkite-agent" "${status}" "${msg}" "job_seconds=${longest}s;$((JOB_WARN_HOURS * 3600));$((JOB_CRIT_HOURS * 3600))"
}

# check_docker_providers probes each Docker context whose daemon socket exists.
# A refused connection means the provider is stopped, which is normal since jobs
# start providers on demand; a probe that times out means a hung daemon.
# Contexts that are up are left in UP_CONTEXTS for check_leftover_containers.
UP_CONTEXTS=""
function check_docker_providers {
  local ctx endpoint sock rc up="" stopped="" hung="" status=${OK}
  if [ -z "${TIMEOUT}" ]; then
    push "docker-providers" "${UNKNOWN}" "Neither gtimeout nor timeout is installed"
    return
  fi
  local contexts
  if ! contexts=$(docker context ls --format '{{.Name}}' 2>&1); then
    push "docker-providers" "${UNKNOWN}" "docker context ls failed: ${contexts}"
    return
  fi
  for ctx in ${contexts}; do
    [ "${ctx}" = "default" ] && continue
    endpoint=$(docker context inspect -f '{{.Endpoints.docker.Host}}' "${ctx}" 2>/dev/null)
    sock=${endpoint#unix://}
    if [ "${sock}" != "${endpoint}" ] && [ ! -S "${sock}" ]; then
      stopped="${stopped} ${ctx}"
      continue
    fi
    # OrbStack's app keeps running with its VM stopped; probing could wake the VM.
    if [ "${ctx}" = "orbstack" ] && ! pgrep -qf "OrbStack Helper vmgr"; then
      stopped="${stopped} ${ctx}"
      continue
    fi
    "${TIMEOUT}" "${PROBE_TIMEOUT}" docker --context "${ctx}" version --format '{{.Server.Version}}' >/dev/null 2>&1 </dev/null
    rc=$?
    if [ "${rc}" -eq 0 ]; then
      up="${up} ${ctx}"
    elif [ "${rc}" -eq 124 ]; then
      hung="${hung} ${ctx}"
      status=${CRITICAL}
    else
      stopped="${stopped} ${ctx}"
    fi
  done
  UP_CONTEXTS=${up}
  push "docker-providers" "${status}" \
    "hung:${hung:- none}; up:${up:- none}; stopped:${stopped:- none}" \
    "up=$(echo "${up}" | wc -w | tr -d ' ')" "hung=$(echo "${hung}" | wc -w | tr -d ' ');;1"
}

# check_leftover_containers reports containers left in any running provider
# while no job runs; they can hold ports a later job needs.
function check_leftover_containers {
  local ctx count total=0 detail=""
  if pgrep -qf "buildkite-agent bootstrap"; then
    push "leftover-containers" "${OK}" "Job running, not checked"
    return
  fi
  for ctx in ${UP_CONTEXTS}; do
    count=$("${TIMEOUT}" "${PROBE_TIMEOUT}" docker --context "${ctx}" ps -aq 2>/dev/null </dev/null | wc -l | tr -d ' ')
    if [ "${count:-0}" -gt 0 ]; then
      total=$((total + count))
      detail="${detail} ${ctx}=${count}"
    fi
  done
  if [ "${total}" -gt 0 ]; then
    push "leftover-containers" "${WARNING}" "${total} container(s) with no job running:${detail}" "containers=${total}"
  else
    push "leftover-containers" "${OK}" "No containers with no job running" "containers=0"
  fi
}

if [ "${DRY_RUN}" != "true" ]; then
  for var in ICINGA_URL ICINGA_API_USER ICINGA_API_PASSWORD; do
    if [ -z "${!var:-}" ]; then
      echo "${var} is not set in ${CONFIG_DIR}/env" >&2
      exit 1
    fi
  done
fi

check_host
check_disk
check_load
check_buildkite_agent
check_docker_providers
check_leftover_containers
