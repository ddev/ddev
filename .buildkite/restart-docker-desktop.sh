#!/usr/bin/env bash
# restart-docker-desktop.sh
#
# Restart Docker Desktop and wait for WSL2 integration to become active in
# the given distro.
#
# Background: Docker Desktop loses WSL2 integration when things like
# apt-get remove docker-ce-cli remove /usr/bin/docker, or when WSLInterop
# is cleared from binfmt_misc. A Docker Desktop restart forces it to
# re-inject its integration binaries into configured distros.
#
# IMPORTANT: Avoid 'docker desktop stop' followed by 'docker desktop start'.
# During the stop window, /mnt/c/ (CAROOT) becomes briefly inaccessible to
# WSL2 processes. DDEV's readCAROOT() silently returns "" in this window,
# causing DDEV to generate a new CA not trusted by Windows — TLS failures
# (see https://github.com/ddev/ddev/issues/8485). Use 'docker desktop restart'
# instead, which Docker Desktop handles as a single atomic operation.
#
# Usage: bash restart-docker-desktop.sh <distro-name>
# Exit:  0  integration confirmed working in <distro-name>
#        1  timed out or unexpected error

set -o pipefail
set -o nounset

DISTRO="${1:?Usage: $0 <distro-name>}"

TIMEOUT_START=180     # seconds to wait for Docker Desktop to report running
TIMEOUT_INTEGRATION=120  # seconds to wait for docker ps to work inside distro

wait_for_docker_desktop_running() {
    local elapsed=0
    while true; do
        local status
        status=$(docker desktop status 2>&1 || true)
        if echo "$status" | grep -qi "Status[[:space:]]*running"; then
            echo "restart-docker-desktop: Docker Desktop running (${elapsed}s elapsed)"
            return 0
        fi
        if [ "$elapsed" -ge "$TIMEOUT_START" ]; then
            echo "restart-docker-desktop: ERROR: timed out after ${TIMEOUT_START}s waiting for running"
            echo "restart-docker-desktop: last status: $status"
            return 1
        fi
        sleep 5
        elapsed=$((elapsed + 5))
    done
}

echo "restart-docker-desktop: state BEFORE restart:"
bash "$(dirname "$0")/dump-docker-desktop-state.sh" "$DISTRO" "before-restart" || true

# Use 'docker desktop restart' as a single atomic operation rather than
# separate stop+start, to minimize the window where CAROOT is inaccessible.
echo "restart-docker-desktop: restarting Docker Desktop to restore WSL2 integration in $DISTRO..."
docker desktop restart || true

wait_for_docker_desktop_running || exit 1

# Wait for both WSL2 integration in the distro AND Windows-side docker ps.
echo "restart-docker-desktop: waiting for WSL2 integration in $DISTRO (up to ${TIMEOUT_INTEGRATION}s)..."
elapsed=0
while true; do
    wsl_ok=false
    win_ok=false
    wsl.exe -d "$DISTRO" -- docker ps >/dev/null 2>&1 && wsl_ok=true
    docker ps >/dev/null 2>&1 && win_ok=true
    if [ "$wsl_ok" = "true" ] && [ "$win_ok" = "true" ]; then
        echo "restart-docker-desktop: WSL2 integration confirmed in $DISTRO and Windows docker ps OK (${elapsed}s elapsed)"
        bash "$(dirname "$0")/dump-docker-desktop-state.sh" "$DISTRO" "integration-restored" || true
        exit 0
    fi
    if [ "$elapsed" -ge "$TIMEOUT_INTEGRATION" ]; then
        echo "restart-docker-desktop: ERROR: timed out after ${TIMEOUT_INTEGRATION}s (wsl_ok=$wsl_ok win_ok=$win_ok)"
        echo "restart-docker-desktop: state at integration timeout (shows whether the symlink, cli-tools mount, or socket is the problem):"
        bash "$(dirname "$0")/dump-docker-desktop-state.sh" "$DISTRO" "integration-timeout" || true
        exit 1
    fi
    echo "restart-docker-desktop: waiting... wsl_ok=$wsl_ok win_ok=$win_ok (${elapsed}s)"
    sleep 10
    elapsed=$((elapsed + 10))
done
