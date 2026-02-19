#!/usr/bin/env bash
#ddev-generated

# cloudflared share provider for DDEV
# Documentation: https://docs.ddev.com/en/stable/users/topics/sharing/
#
# To customize: remove the 'ddev-generated' line above and edit as needed.
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
if [[ "${DDEV_DEBUG:-}" == "true" ]] || [[ "${DDEV_VERBOSE:-}" == "true" ]]; then
  set -x
fi

# Validate cloudflared is installed
if ! command -v cloudflared >/dev/null 2>&1; then
  echo "Error: cloudflared not found in PATH." >&2
  echo "Install from https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/" >&2
  exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
  echo "Error: DDEV_LOCAL_URL not set" >&2
  exit 1
fi

# Start cloudflared and capture output
echo "Starting cloudflared tunnel..." >&2

# Parse args to determine tunnel mode
# --tunnel and --hostname are DDEV-specific and stripped before passing to cloudflared
ARGS="${DDEV_SHARE_ARGS:-}"
TUNNEL_NAME=""
HOSTNAME=""

# Check for named tunnel mode (--tunnel flag present)
# Supports both --tunnel my-tunnel and --tunnel=my-tunnel
if [[ "$ARGS" =~ --tunnel([[:space:]]+|=)([^[:space:]]+) ]]; then
  TUNNEL_NAME="${BASH_REMATCH[2]}"
  ARGS=$(echo "$ARGS" | sed -E 's/--tunnel([[:space:]]+|=)[^[:space:]]+//')
fi

# Extract hostname if present (for URL output only - not passed to cloudflared)
# Supports both --hostname mysite.example.com and --hostname=mysite.example.com
if [[ "$ARGS" =~ --hostname([[:space:]]+|=)([^[:space:]]+) ]]; then
  HOSTNAME="${BASH_REMATCH[2]}"
  ARGS=$(echo "$ARGS" | sed -E 's/--hostname([[:space:]]+|=)[^[:space:]]+//')
fi

# Normalize whitespace left over from stripping flags
ARGS=$(echo "$ARGS" | sed -E 's/[[:space:]]+/ /g;s/^ //;s/ $//')

# For named tunnels with known hostname, output URL immediately
URL_FOUND=""
if [[ -n "$HOSTNAME" ]]; then
  echo "https://$HOSTNAME" # Output to stdout - CRITICAL: This is captured by DDEV
  URL_FOUND="https://$HOSTNAME"
fi

# Build and run the cloudflared command, piping all output through the reader
# Using --protocol http2 for better compatibility
if [[ -n "$TUNNEL_NAME" ]]; then
  echo "Using named tunnel: $TUNNEL_NAME" >&2
  echo >&2
  echo "Running command: cloudflared --url $DDEV_LOCAL_URL --protocol http2${ARGS:+ $ARGS} tunnel run $TUNNEL_NAME" >&2
  echo >&2
  cloudflared --url "$DDEV_LOCAL_URL" --protocol http2 $ARGS tunnel run "$TUNNEL_NAME" 2>&1
else
  echo >&2
  echo "Running command: cloudflared tunnel --url $DDEV_LOCAL_URL --protocol http2 $ARGS" >&2
  echo >&2
  cloudflared tunnel --url "$DDEV_LOCAL_URL" --protocol http2 $ARGS 2>&1
fi | while IFS= read -r line; do
  # For quick tunnels, look for the cloudflared URL in the output
  # Exclude api.trycloudflare.com which appears in error messages
  if [[ -z "$URL_FOUND" ]] && [[ "$line" =~ https://[a-z0-9-]+\.trycloudflare\.com ]]; then
    POTENTIAL_URL="${BASH_REMATCH[0]}"
    if [[ ! "$POTENTIAL_URL" =~ api\.trycloudflare\.com ]]; then
      URL_FOUND="$POTENTIAL_URL"
      echo "$URL_FOUND" # Output to stdout - CRITICAL: This is captured by DDEV
    fi
  fi
  # Show non-info output to user (warnings, errors, etc.); show all in verbose mode
  if [[ "${DDEV_VERBOSE:-}" == "true" ]] || [[ ! "$line" =~ " INF " ]]; then
    echo "$line" >&2
  fi
done
