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

# Start cloudflared in background and capture output
echo "Starting cloudflared tunnel..." >&2
TUNNEL_LOG=$(mktemp)
trap "rm -f $TUNNEL_LOG" EXIT

cloudflared tunnel --url "$DDEV_LOCAL_URL" ${DDEV_SHARE_CLOUDFLARED_ARGS:-} 2>&1 | tee "$TUNNEL_LOG" | grep -v "^[0-9]" >&2 &
CF_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $CF_PID 2>/dev/null; then
        kill $CF_PID 2>/dev/null || true
    fi
    rm -f "$TUNNEL_LOG"
}
trap cleanup EXIT

# Wait for cloudflared to output the tunnel URL
URL=""
for i in {1..30}; do
    # Look for the "Your quick Tunnel has been created!" message and extract URL
    URL=$(grep -oE "https://[a-z0-9-]+\.trycloudflare\.com" "$TUNNEL_LOG" | tail -1)

    if [[ -n "$URL" ]]; then
        echo "$URL"  # Output to stdout - CRITICAL: This is captured by DDEV
        break
    fi
    sleep 1
done

if [[ -z "$URL" ]]; then
    echo "Error: Failed to get cloudflared URL after 30 seconds" >&2
    cat "$TUNNEL_LOG" >&2
    exit 1
fi

# Wait for cloudflared to exit
wait $CF_PID
