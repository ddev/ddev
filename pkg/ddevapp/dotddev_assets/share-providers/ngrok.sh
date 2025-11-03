#!/bin/bash
#ddev-generated

# ngrok share provider for DDEV
# Documentation: https://ddev.readthedocs.io/en/stable/users/topics/sharing/
#
# To customize: remove the '#ddev-generated' line above and edit as needed.
# To create a variant: copy to a new file like my-ngrok.sh

set -euo pipefail

# Validate ngrok is installed
if ! command -v ngrok &> /dev/null; then
    echo "Error: ngrok not found in PATH. Install from https://ngrok.com/download" >&2
    exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
    echo "Error: DDEV_LOCAL_URL not set" >&2
    exit 1
fi

# Start ngrok in background
ngrok http "$DDEV_LOCAL_URL" ${DDEV_SHARE_NGROK_ARGS:-} &
NGROK_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $NGROK_PID 2>/dev/null; then
        kill $NGROK_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Poll ngrok API for public URL (30 second timeout)
echo "Starting ngrok tunnel..." >&2
URL=""
for i in {1..30}; do
    URL=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null | \
          jq -r '.tunnels[0].public_url' 2>/dev/null || echo "")

    if [[ -n "$URL" && "$URL" != "null" ]]; then
        echo "$URL"  # Output to stdout - CRITICAL: This is captured by DDEV
        break
    fi
    sleep 1
done

if [[ -z "$URL" || "$URL" == "null" ]]; then
    echo "Error: Failed to get ngrok URL after 30 seconds" >&2
    exit 1
fi

# Wait for ngrok to exit
wait $NGROK_PID
