#!/usr/bin/env bash
# Post-create command for DDEV devcontainer feature
set -eu -o pipefail

# Ensure /workspaces is writable for config storage
if [ -d /workspaces ]; then
    sudo chmod ugo+w /workspaces 2>/dev/null || true
fi

# Verify DDEV installation
ddev version && echo 'DDEV is ready to use'
