#!/usr/bin/env bash

# Check a testbot or test environment to make sure it's likely to be sane.
# We should add to this script whenever a testbot fails and we can figure out why.

MIN_DDEV_VERSION=v1.24.0

set -o errexit
set -o pipefail
set -o nounset

# thanks to https://stackoverflow.com/a/24067243/215713
function version_gt() { test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"; }

DISK_AVAIL=$(df -k . | awk '/[0-9]%/ { gsub(/%/, ""); print $5}')
if [ ${DISK_AVAIL} -ge 95 ] ; then
    echo "Disk usage is ${DISK_AVAIL}% on $(hostname), not usable";
    exit 1;
else
   echo "Disk usage is ${DISK_AVAIL}% on $(hostname).";
fi

# Determine whether this job uses the Windows host's Docker Desktop. The WSL2
# docker-ce (and docker-inside) installer cases run Docker CE *inside* the WSL2
# distro and never touch the host's Docker Desktop — so they must not wait for
# or start it, and the host docker sanity checks below would only test (and
# block on) an unrelated, possibly-stopped Docker Desktop. Worse, starting
# Docker Desktop for a docker-ce job makes its WSL2 integration fight the
# in-distro docker-ce daemon over /var/run/docker.sock. Detected from the
# installer pipeline's INSTALLER_CASE; when unset (other pipelines) prior
# behavior is preserved.
USES_HOST_DOCKER=true
case "${INSTALLER_CASE:-}" in
    *-ce|*-inside|ps1-docker-inside) USES_HOST_DOCKER=false ;;
esac

# On Windows with Docker Desktop, CHECK that Docker Desktop's frontend is running.
# sanetestbot only *checks* the runner; starting Docker Desktop is the job of
# start-docker-desktop.sh, which must run earlier in the pipeline. On Linux and
# macOS Docker runs as a daemon and is expected to already be up; this block is
# Windows-only.
if [ "$(go env GOOS)" = "windows" ] && [ "$USES_HOST_DOCKER" = "true" ]; then
    # Check 'docker desktop status' (the frontend), not 'docker ps' (the Engine
    # service). On Windows the Docker Engine service (SYSTEM) keeps running even
    # when Docker Desktop.exe (the frontend) is stopped — so docker ps succeeds
    # even when Docker Desktop is down. WSL2 integration is managed by the
    # frontend, so we must check 'docker desktop status'. A few retries absorb a
    # just-started Docker Desktop still settling; we do NOT start it here.
    dd_frontend_ok=false
    for _retry in 1 2 3 4 5 6; do
        dd_status=$(docker desktop status 2>&1 || true)
        if echo "$dd_status" | grep -qi "Status[[:space:]]*running"; then
            dd_frontend_ok=true
            echo "$(date -u +%H:%M:%S) Docker Desktop frontend running (check $_retry)"
            break
        fi
        echo "$(date -u +%H:%M:%S) Docker Desktop not running (check $_retry, status: $dd_status), retrying in 10s..."
        sleep 10
    done

    if [ "$dd_frontend_ok" = "false" ]; then
        dd_status=$(docker desktop status 2>&1 || true)
        echo "ERROR: Docker Desktop frontend is not running (status: $dd_status)."
        echo "start-docker-desktop.sh should have started it earlier in the pipeline."
        exit 1
    fi
fi

# Host docker sanity checks. These exercise the host's docker daemon (Docker
# Desktop on Windows, the local daemon on macOS/Linux). Skip them for WSL2
# docker-ce/-inside installer cases, which use Docker CE inside the distro and
# do not use the host's Docker Desktop — running them would only block on an
# unrelated, possibly-stopped Docker Desktop.
if [ "$USES_HOST_DOCKER" = "true" ]; then
    # Test to make sure docker is installed and working.
    # If it doesn't become ready then we just keep this testbot occupied :)
    docker ps >/dev/null
    while ! docker ps >/dev/null 2>&1 ; do
        echo "Waiting for docker to be ready $(date)"
        sleep 60
    done

    # Test that docker can allocate 80 and 443, get ddev/ddev-utilities
    docker pull ddev/ddev-utilities >/dev/null
    # Try the docker run command twice because of the really annoying mkdir /c: file exists bug
    # Apparently https://github.com/docker/for-win/issues/1560
    (sleep 1 && (docker run --rm -t -p 80:80 -p 443:443 -p 1081:1081 -p 1082:1082 -v /$HOME:/tmp/junker99 ddev/ddev-utilities ls //tmp/junker99 >/dev/null) || (sleep 1 && docker run --rm -t -p 80:80 -p 443:443 -p 1081:1081 -p 1082:1082 -v /$HOME:/tmp/junker99 ddev/ddev-utilities ls //tmp/junker99 >/dev/null ))
else
    echo "Skipping host Docker Desktop checks for INSTALLER_CASE=${INSTALLER_CASE:-} (uses Docker CE inside WSL2, not the host Docker Desktop)"
fi

# Check that required commands are available.
for command in git go make mysql ngrok; do
    command -v $command >/dev/null || ( echo "Did not find command installed '$command'" && exit 2 )
done

if [ "$(go env GOOS)" = "windows"  -a "$(git config core.autocrlf)" != "false" ] ; then
 echo "git config core.autocrlf is not set to false on windows"
 exit 3
fi

# On Windows/WSL2: ensure the binfmt_misc WSLInterop entry is registered in each test distro.
#
# Background: WSL2 uses a binfmt_misc entry named "WSLInterop" so that Linux shells can
# transparently invoke Windows .exe binaries. This entry is lost whenever docker-ce is
# removed from a distro — the package's post-remove scripts clear binfmt_misc entries.
# It is also lost when systemd remounts binfmt_misc during boot before wsl.conf [boot]
# command has run. When the entry is absent, any .exe called from within the distro fails
# with "cannot execute binary file: Exec format error".
#
# Consequence for these tests: the PS1 install scripts call `wsl -d <distro> mkcert.exe`
# to add the mkcert CA to the Windows certificate store. If interop is broken, that call
# silently fails, the CA is never trusted by Windows, and every subsequent PowerShell
# HTTPS check fails with an SSL/TLS error — a hard-to-diagnose failure miles from the root cause.
#
# wsl-fix-interop re-registers the entry by writing the magic string to
# /proc/sys/fs/binfmt_misc/register. It is idempotent (exits 0 if already present).
# Requires a one-time installation per distro — see buildkite-testmachine-setup.md.
# See https://github.com/rfay/wsl-fix-interop
if [ "$(go env GOOS)" = "windows" ]; then
    for distro in ddev-test-ubuntu-ce ddev-test-ubuntu-desktop ddev-test-ubuntu2404-ce ddev-test-ubuntu2404-desktop ddev-test-debian-ce ddev-test-debian-desktop; do
        fix_out=$(wsl.exe -d "$distro" bash -c "sudo wsl-fix-interop" 2>&1) \
            && echo "wsl-fix-interop in $distro: $fix_out" \
            || echo "WARNING: wsl-fix-interop failed or not installed in $distro (skipping)"
    done

fi

echo "-- testbot $HOSTNAME seems to be set up OK --"
