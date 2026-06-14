#!/bin/bash
# Wait up to N seconds for systemd to be ready as PID 1 in this distro.
# Considers any of running|degraded|starting as "ready enough" — the
# installer can proceed once systemd is past basic.target so service
# units can be started. Exits 0 on ready, 1 on timeout.
#
# Usage: wait_for_systemd.sh [timeout_seconds]   (default 15)

set -u

TIMEOUT="${1:-15}"
STATE=""

for _ in $(seq 1 "$TIMEOUT"); do
    STATE=$(systemctl is-system-running 2>/dev/null || true)
    case "$STATE" in
        running|degraded|starting)
            echo "systemd state: $STATE"
            exit 0
            ;;
    esac
    sleep 1
done

echo "systemd state: '${STATE:-unknown}' (timed out after ${TIMEOUT}s)"
exit 1
