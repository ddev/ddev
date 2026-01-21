#!/usr/bin/env bash
# Post-create command for DDEV devcontainer feature
set -eu -o pipefail

# Ensure /workspaces is writable for config storage
if [ -d /workspaces ]; then
    sudo chmod ugo+w /workspaces 2>/dev/null || true
fi

# Check if DDEV should be built from source (for development)
if [ -n "${DDEV_BUILD_FROM_SOURCE:-}" ] && [ -d "${DDEV_BUILD_FROM_SOURCE}" ]; then
    echo "Building DDEV from source at ${DDEV_BUILD_FROM_SOURCE}"
    cd "${DDEV_BUILD_FROM_SOURCE}"
    ARCH=$(dpkg --print-architecture)
    make
    sudo cp ".gotmp/bin/linux_${ARCH}/ddev" /usr/local/bin/ddev
    sudo chmod +x /usr/local/bin/ddev
    echo "Development build installed for linux_${ARCH}"
fi

# Verify DDEV installation
ddev version && echo 'DDEV is ready to use'
