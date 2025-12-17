#!/usr/bin/env bash
#ddev-generated

# cloudflared share provider for DDEV
# Documentation: https://ddev.readthedocs.io/en/stable/users/topics/sharing/
#
# To customize: remove the '#ddev-generated' line above and edit as needed.
#
# Quick tunnel (default): Creates a temporary URL like https://xxx.trycloudflare.com
#   No configuration needed - just run `ddev share --provider=cloudflared`
#
# Named tunnel (custom domain): Use your own domain managed by Cloudflare
#   1. Create tunnel: cloudflared tunnel create my-tunnel
#   2. Add DNS: cloudflared tunnel route dns my-tunnel mysite.example.com
#   3. Configure: ddev config --share-provider-args="--tunnel my-tunnel --hostname mysite.example.com"
#      (--hostname tells DDEV what URL to display; DNS routing is done in step 2)
#   4. Run: ddev share --provider=cloudflared

set -euo pipefail

# Enable debug output if DDEV_DEBUG or DDEV_VERBOSE is set
VERBOSE=""
if [[ "${DDEV_DEBUG:-}" == "true" ]] || [[ "${DDEV_VERBOSE:-}" == "true" ]]; then
    set -x
    VERBOSE="true"
fi

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

# Use a named pipe to capture stderr while also reading it
PIPE=$(mktemp -u)
mkfifo "$PIPE"
trap "rm -f $PIPE" EXIT

# Parse args to determine tunnel mode
ARGS="${DDEV_SHARE_ARGS:-}"
TUNNEL_NAME=""
HOSTNAME=""
OTHER_ARGS=""

# Check for named tunnel mode (--tunnel flag present)
if [[ "$ARGS" =~ --tunnel[[:space:]]+([^[:space:]]+) ]]; then
    TUNNEL_NAME="${BASH_REMATCH[1]}"
    # Remove --tunnel and its value from args
    OTHER_ARGS=$(echo "$ARGS" | sed -E 's/--tunnel[[:space:]]+[^[:space:]]+//')
fi

# Extract hostname if present (for URL output only - not passed to cloudflared)
if [[ "$ARGS" =~ --hostname[[:space:]]+([^[:space:]]+) ]]; then
    HOSTNAME="${BASH_REMATCH[1]}"
    # Remove --hostname from OTHER_ARGS (it's only for DDEV URL display, not cloudflared CLI)
    OTHER_ARGS=$(echo "$OTHER_ARGS" | sed -E 's/--hostname[[:space:]]+[^[:space:]]+//')
fi

# Build and run the cloudflared command
# Using --protocol http2 for better compatibility
if [[ -n "$TUNNEL_NAME" ]]; then
    # Named tunnel mode: cloudflared --url <url> --protocol http2 tunnel run <name>
    # Note: --url and --protocol must come BEFORE "tunnel run"
    echo "Using named tunnel: $TUNNEL_NAME" >&2
    cloudflared --url "$DDEV_LOCAL_URL" --protocol http2 tunnel run "$TUNNEL_NAME" 2> "$PIPE" &
else
    # Quick tunnel mode (default): cloudflared tunnel --url <url> --protocol http2 [args]
    cloudflared tunnel --url "$DDEV_LOCAL_URL" --protocol http2 $ARGS 2> "$PIPE" &
fi
CF_PID=$!

# Function to cleanup on exit
cleanup() {
    if kill -0 $CF_PID 2>/dev/null; then
        kill $CF_PID 2>/dev/null || true
    fi
    rm -f "$PIPE"
}
trap cleanup EXIT

# For named tunnels with known hostname, output URL immediately
# For quick tunnels, we need to wait for cloudflared to report the URL
if [[ -n "$HOSTNAME" ]]; then
    # Named tunnel with known hostname - output URL now
    echo "https://$HOSTNAME"  # Output to stdout - CRITICAL: This is captured by DDEV
    URL="https://$HOSTNAME"
fi

# Read from pipe and extract URL (for quick tunnels) or just forward output
URL_FOUND="${URL:-}"
while IFS= read -r line; do
    # For quick tunnels, look for the cloudflared URL in the output
    if [[ -z "$URL_FOUND" ]] && [[ "$line" =~ https://[a-z0-9-]+\.trycloudflare\.com ]]; then
        URL_FOUND="${BASH_REMATCH[0]}"
        echo "$URL_FOUND"  # Output to stdout - CRITICAL: This is captured by DDEV
    fi
    # In verbose mode, show all output; otherwise only show errors/warnings
    if [[ -n "$VERBOSE" ]]; then
        echo "$line" >&2
    elif [[ "$line" =~ ^[0-9T:-]+Z\ (ERR|WRN|FTL) ]] && [[ ! "$line" =~ "Cannot determine default origin certificate" ]]; then
        # Only show error messages and warnings to user, suppress verbose INFO logs
        # Skip benign errors about origin certificate (not needed for quick tunnels)
        echo "$line" >&2
    fi
done < "$PIPE" &
READER_PID=$!

# Wait for cloudflared to exit
wait $CF_PID
EXIT_CODE=$?

# Clean up reader process
kill $READER_PID 2>/dev/null || true
wait $READER_PID 2>/dev/null || true

exit $EXIT_CODE
