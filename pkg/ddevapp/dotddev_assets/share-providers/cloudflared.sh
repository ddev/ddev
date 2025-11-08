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

# Start cloudflared and capture output
echo "Starting cloudflared tunnel..." >&2

# Use a temp file to store URL when found by background reader
URL_FILE=$(mktemp)
trap "rm -f $URL_FILE" EXIT

# Use a named pipe to capture stderr while also reading it
PIPE=$(mktemp -u)
mkfifo "$PIPE"
trap "rm -f $PIPE $URL_FILE" EXIT

# Start cloudflared with stderr to pipe
cloudflared tunnel --url "$DDEV_LOCAL_URL" ${DDEV_SHARE_CLOUDFLARED_ARGS:-} 2> "$PIPE" &
CF_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $CF_PID 2>/dev/null; then
        kill $CF_PID 2>/dev/null || true
    fi
    rm -f "$PIPE" "$URL_FILE"
}
trap cleanup EXIT

# Read from pipe and extract URL (background process)
while IFS= read -r line; do
    # Look for the cloudflared URL in the output
    if [[ ! -s "$URL_FILE" ]] && [[ "$line" =~ https://[a-z0-9-]+\.trycloudflare\.com ]]; then
        # Store URL in temp file instead of outputting immediately
        echo "${BASH_REMATCH[0]}" > "$URL_FILE"
        echo "Tunnel URL found, verifying connectivity..." >&2
    fi
    # Continue reading and sending to stderr for logging
    echo "$line" >&2
done < "$PIPE" &
READER_PID=$!

# Wait for URL to be found (max 30 seconds)
for i in {1..30}; do
    if [[ -s "$URL_FILE" ]]; then
        URL=$(cat "$URL_FILE")
        break
    fi
    sleep 1
done

if [[ -z "$URL" ]]; then
    echo "Error: Failed to get cloudflared URL after 30 seconds" >&2
    kill $READER_PID 2>/dev/null || true
    exit 1
fi

# Verify tunnel is actually functional before returning URL
# Cloudflare may print the URL before the tunnel is fully connected
echo "Waiting for tunnel to be ready..." >&2
for i in {1..15}; do
    # Try a HEAD request to verify tunnel responds
    if curl -sf --head --max-time 2 "$URL" >/dev/null 2>&1; then
        echo "Tunnel is ready!" >&2
        echo "$URL"  # Output to stdout - CRITICAL: This is captured by DDEV
        break
    fi
    if [[ $i -eq 15 ]]; then
        echo "Warning: Tunnel URL found but not responding after 15 seconds" >&2
        echo "Cloudflare may still be establishing the connection" >&2
        echo "$URL"  # Output URL anyway, let user decide
    fi
    sleep 1
done

# Wait for cloudflared to exit
wait $CF_PID
EXIT_CODE=$?

# Clean up reader process
kill $READER_PID 2>/dev/null || true
wait $READER_PID 2>/dev/null || true

exit $EXIT_CODE
