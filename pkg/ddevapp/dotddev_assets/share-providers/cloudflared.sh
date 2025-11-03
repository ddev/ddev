#!/bin/bash
#ddev-generated

# cloudflared share provider for DDEV
# Documentation: https://ddev.readthedocs.io/en/stable/users/topics/sharing/
#
# To customize: remove the '#ddev-generated' line above and edit as needed.

set -euo pipefail

# Validate cloudflared is installed
if ! command -v cloudflared &> /dev/null; then
    echo "Error: cloudflared not found in PATH." >&2
    echo "Install from https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation" >&2
    exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
    echo "Error: DDEV_LOCAL_URL not set" >&2
    exit 1
fi

# Start cloudflared in background
cloudflared tunnel --url "$DDEV_LOCAL_URL" ${DDEV_SHARE_CLOUDFLARED_ARGS:-} &
CF_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $CF_PID 2>/dev/null; then
        kill $CF_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# cloudflared exposes metrics API on random port 20241-20245
# Poll all possible ports for tunnel URL
echo "Starting cloudflared tunnel..." >&2
HOSTNAME=""
for i in {1..30}; do
    for PORT in {20241..20245}; do
        HOSTNAME=$(curl -s "http://127.0.0.1:$PORT/quicktunnel" 2>/dev/null | \
                   jq -r '.hostname' 2>/dev/null || echo "")

        if [[ -n "$HOSTNAME" && "$HOSTNAME" != "null" ]]; then
            echo "https://$HOSTNAME"  # Output to stdout - CRITICAL: This is captured by DDEV
            break 2
        fi
    done
    sleep 1
done

if [[ -z "$HOSTNAME" || "$HOSTNAME" == "null" ]]; then
    echo "Error: Failed to get cloudflared URL after 30 seconds" >&2
    exit 1
fi

# Wait for cloudflared to exit
wait $CF_PID
