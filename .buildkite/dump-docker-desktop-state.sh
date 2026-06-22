#!/usr/bin/env bash
# dump-docker-desktop-state.sh [distro] [label]
#
# One-shot diagnostic dump of Docker Desktop / WSL2 integration state on a
# Windows test runner. Used to understand Docker Desktop start/stop behavior and
# WSL2-integration loss across Buildkite jobs. Best-effort: never fails (always
# exits 0), every probe is guarded, so it is safe to call from a trap or
# anywhere else.
#
# Usage:
#   bash dump-docker-desktop-state.sh [distro] [label]

# Disable git-bash POSIX->Windows path conversion so the slash-flags below
# (tasklist.exe //FI, schtasks.exe //Query) and the in-distro Linux paths inside
# the bash -c strings are passed through verbatim.
export MSYS_NO_PATHCONV=1

DISTRO="${1:-}"
LABEL="${2:-}"

echo "===== dump-docker-desktop-state ${LABEL:+[$LABEL] }$(date '+%H:%M:%S') ====="

echo -n "docker desktop status:   "
docker desktop status 2>&1 | grep -iE "Status|Could not" | head -1 || echo "(error)"

echo -n "host docker ps:          "
if docker ps --format "{{.Names}}" 2>/dev/null | tr '\n' ' '; then echo "(ok)"; else echo "(FAILED)"; fi

echo "Docker Desktop-related Windows processes (name / PID):"
tasklist.exe //FI "IMAGENAME eq Docker Desktop.exe" //FO CSV //NH 2>/dev/null | grep -i "docker" | sed 's/^/  /' || true
tasklist.exe //FI "IMAGENAME eq com.docker.backend.exe" //FO CSV //NH 2>/dev/null | grep -i "docker" | sed 's/^/  /' || true
tasklist.exe //FI "IMAGENAME eq com.docker.service" //FO CSV //NH 2>/dev/null | grep -i "docker" | sed 's/^/  /' || true

echo -n "scheduled task DDEVStartDockerDesktopDetached: "
schtasks.exe //Query //TN "DDEVStartDockerDesktopDetached" //FO LIST 2>/dev/null \
    | grep -iE "Status|Last Run Time|Last Result" | tr '\n' ' ' || echo -n "(not registered)"
echo ""

echo "WSL running distros:"
wsl.exe -l --running 2>/dev/null | tr -d '\0' | sed 's/^/  /' || echo "  (wsl error)"

if [ -n "$DISTRO" ]; then
    echo -n "distro[$DISTRO] docker ps:        "
    if wsl.exe -d "$DISTRO" -- docker ps --format "{{.Names}}" 2>/dev/null | tr '\n' ' '; then echo "(ok)"; else echo "(FAILED)"; fi

    echo -n "distro[$DISTRO] /usr/bin/docker:   "
    wsl.exe -d "$DISTRO" -- bash -c \
        'if [ -L /usr/bin/docker ]; then t=$(readlink /usr/bin/docker); if [ -x /usr/bin/docker ]; then echo "symlink -> $t (TARGET OK)"; else echo "symlink -> $t (TARGET BROKEN)"; fi; elif [ -f /usr/bin/docker ]; then echo "regular file (docker-ce-cli)"; else echo "MISSING"; fi' 2>/dev/null || echo "(distro error)"

    echo -n "distro[$DISTRO] cli-tools binary:  "
    wsl.exe -d "$DISTRO" -- bash -c \
        'f=/mnt/wsl/docker-desktop/cli-tools/usr/bin/docker; if [ -x "$f" ]; then echo "EXISTS and executable"; elif [ -f "$f" ]; then echo "exists but not executable"; elif [ -d /mnt/wsl/docker-desktop/cli-tools ]; then echo "cli-tools dir exists but docker MISSING"; elif [ -d /mnt/wsl/docker-desktop ]; then echo "mount exists but cli-tools dir MISSING"; else echo "mount NOT PRESENT"; fi' 2>/dev/null || echo "(distro error)"

    echo -n "distro[$DISTRO] docker.sock:       "
    wsl.exe -d "$DISTRO" -- bash -c '[ -S /var/run/docker.sock ] && echo "exists" || echo "MISSING"' 2>/dev/null || echo "(distro error)"

    echo -n "distro[$DISTRO] WSLInterop:        "
    wsl.exe -d "$DISTRO" -- bash -c '[ -f /proc/sys/fs/binfmt_misc/WSLInterop ] && grep -q enabled /proc/sys/fs/binfmt_misc/WSLInterop 2>/dev/null && echo "registered+enabled" || echo "MISSING/disabled"' 2>/dev/null || echo "(distro error)"
fi

echo "===== end dump-docker-desktop-state ${LABEL:+[$LABEL]} ====="
exit 0
