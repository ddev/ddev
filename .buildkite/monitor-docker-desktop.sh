#!/usr/bin/env bash
# monitor-docker-desktop.sh
#
# Continuously shows Docker Desktop health on the Windows host.
# Run from Git Bash on the test machine:
#   bash .buildkite/monitor-docker-desktop.sh
#   bash .buildkite/monitor-docker-desktop.sh ddev-test-ubuntu-desktop  # custom distro

DISTRO="${1:-ddev-test-ubuntu-desktop}"

while true; do
    echo "=== $(date '+%H:%M:%S') ==="

    echo -n "docker desktop status:   "
    docker desktop status 2>&1 | grep -E "Status|Could not" | head -1 || echo "(error)"

    echo -n "host docker ps:          "
    if docker ps --format "{{.Names}}" 2>/dev/null | tr '\n' ' '; then
        echo "(ok)"
    else
        echo "(FAILED)"
    fi

    echo -n "distro docker ps:        "
    wsl.exe -d "$DISTRO" -- docker ps --format "{{.Names}}" 2>/dev/null | tr '\n' ' ' \
        && echo "(ok)" || echo "(FAILED)"

    echo -n "/usr/bin/docker:         "
    wsl.exe -d "$DISTRO" -- bash -c \
        "if [ -L /usr/bin/docker ]; then \
           target=\$(readlink /usr/bin/docker); \
           if [ -x /usr/bin/docker ]; then echo \"symlink -> \$target (TARGET OK)\"; \
           else echo \"symlink -> \$target (TARGET BROKEN)\"; fi; \
         elif [ -f /usr/bin/docker ]; then echo \"regular file (docker-ce-cli)\"; \
         else echo \"MISSING\"; fi" 2>/dev/null || echo "(distro error)"

    echo -n "cli-tools docker binary: "
    wsl.exe -d "$DISTRO" -- bash -c \
        "f=/mnt/wsl/docker-desktop/cli-tools/usr/bin/docker; \
         if [ -x \"\$f\" ]; then echo 'EXISTS and executable'; \
         elif [ -f \"\$f\" ]; then echo 'exists but not executable'; \
         elif [ -d /mnt/wsl/docker-desktop/cli-tools ]; then echo 'cli-tools dir exists but docker MISSING'; \
         elif [ -d /mnt/wsl/docker-desktop ]; then echo 'mount exists but cli-tools dir MISSING'; \
         else echo 'mount NOT PRESENT'; fi" 2>/dev/null || echo "(distro error)"

    echo -n "/var/run/docker.sock:    "
    wsl.exe -d "$DISTRO" -- bash -c \
        "[ -S /var/run/docker.sock ] && echo 'exists' || echo 'MISSING'" 2>/dev/null || echo "(distro error)"

    echo ""
    sleep 3
done
