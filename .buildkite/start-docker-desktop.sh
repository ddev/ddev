#!/usr/bin/env bash
# start-docker-desktop.sh
#
# Test-runner SETUP for Windows: make sure Docker Desktop's frontend is running
# before the docker checks/tests run. This is the Windows analogue of the
# lima/colima/orbstack provider startup in test.sh — it is provider *setup*, not
# a sanity check. (sanetestbot.sh only *checks* the runner; it must not start
# anything.) Call this early, before the docker readiness wait and before
# sanetestbot.sh.
#
# Only acts when this job uses the host's Docker Desktop. The WSL2 docker-ce /
# docker-inside cases run Docker CE inside the distro and do not use the host
# Docker Desktop, so they are skipped (detected from INSTALLER_CASE / DOCKER_TYPE).
#
# Starts Docker Desktop DETACHED via Start-Process so it survives the Buildkite
# job's Windows Job Object, then waits for it to report running. Best-effort:
# always exits 0 — the downstream docker readiness wait / sanetestbot.sh check is
# the gate that fails the job if Docker Desktop never comes up.

set -o pipefail

# Non-Windows runners manage their docker provider elsewhere (test.sh's
# lima/colima/orbstack block, or a daemon). Nothing to do here.
[ "$(go env GOOS)" = "windows" ] || exit 0

# Skip cases that run Docker CE inside the WSL2 distro and don't use the host
# Docker Desktop.
case "${INSTALLER_CASE:-}" in
    *-ce | *-inside | ps1-docker-inside)
        echo "start-docker-desktop: INSTALLER_CASE=${INSTALLER_CASE} uses Docker CE inside WSL2; not starting Docker Desktop"
        exit 0
        ;;
esac
case "${DOCKER_TYPE:-}" in
    wsl2dockerinside | docker-ce)
        echo "start-docker-desktop: DOCKER_TYPE=${DOCKER_TYPE} uses Docker CE inside WSL2; not starting Docker Desktop"
        exit 0
        ;;
esac

if docker desktop status 2>&1 | grep -qi "Status[[:space:]]*running"; then
    echo "start-docker-desktop: Docker Desktop already running."
    exit 0
fi

echo "$(date -u +%H:%M:%S) start-docker-desktop: Docker Desktop not running — starting it (detached)..."
# Use Start-Process to detach Docker Desktop.exe from the Buildkite job's Windows
# Job Object so it survives after the job ends. Never use 'docker desktop stop'
# here — stopping truncates Docker Desktop's per-distro proxy binary and breaks
# WSL2 integration (see git history). This script only ever STARTS.
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command \
    'Start-Process -FilePath "$env:PROGRAMFILES\Docker\Docker\Docker Desktop.exe" -PassThru | Out-Null' 2>/dev/null \
    || docker desktop start || true

elapsed=0
while ! docker desktop status 2>&1 | grep -qi "Status[[:space:]]*running"; do
    if [ "$elapsed" -ge 180 ]; then
        echo "start-docker-desktop: WARNING: Docker Desktop did not report running within ${elapsed}s (downstream checks will catch this)"
        exit 0
    fi
    status=$(docker desktop status 2>&1 || true)
    echo "$(date -u +%H:%M:%S) start-docker-desktop: waiting for Docker Desktop to start (${elapsed}s, status: $status)..."
    sleep 10
    elapsed=$((elapsed + 10))
done
echo "$(date -u +%H:%M:%S) start-docker-desktop: Docker Desktop is running (${elapsed}s)."
exit 0
