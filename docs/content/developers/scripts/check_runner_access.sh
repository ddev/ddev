#!/usr/bin/env bash

# Checks from a maintainer's machine that a test runner's remote access ports
# answer over the tailnet.
# Usage: check_runner_access.sh <runner-name> macos|windows

set -u -o pipefail

usage="Usage: $0 <runner-name> macos|windows"
runner=${1:?${usage}}
case "${2:-}" in
  macos) ports="22 5900" ;;
  windows) ports="22 3389" ;;
  *) echo "${usage}" >&2 && exit 1 ;;
esac

# Exits nonzero for a relayed (DERP) path, which still works, so don't stop on it
tailscale ping -c 1 "${runner}"

status=0
for port in ${ports}; do
  if out=$(nc -vz -w 5 "${runner}" "${port}" 2>&1); then
    echo "${port}: open"
  else
    status=1
    case "${out}" in
      *refused*) echo "${port}: refused, nothing is listening on the runner" ;;
      *) echo "${port}: no answer, blocked by the access policy or the runner's firewall" ;;
    esac
  fi
done
exit ${status}
