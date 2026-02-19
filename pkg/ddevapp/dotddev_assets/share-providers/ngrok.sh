#!/usr/bin/env bash
#ddev-generated

# ngrok share provider for DDEV
# Documentation: https://docs.ddev.com/en/stable/users/topics/sharing/
#
# To customize: remove the 'ddev-generated' line above and edit as needed.
# To create a variant: copy to a new file like my-ngrok.sh

set -euo pipefail

# Enable debug output if DDEV_DEBUG or DDEV_VERBOSE is set
if [[ "${DDEV_DEBUG:-}" == "true" ]] || [[ "${DDEV_VERBOSE:-}" == "true" ]]; then
  set -x
fi

# Validate ngrok is installed
if ! command -v ngrok >/dev/null 2>&1; then
  echo "Error: ngrok not found in PATH. Install from https://ngrok.com/download" >&2
  exit 1
fi

# Validate required environment variables
if [[ -z "${DDEV_LOCAL_URL:-}" ]]; then
  echo "Error: DDEV_LOCAL_URL not set" >&2
  exit 1
fi

# Start ngrok and capture output
echo "Starting ngrok tunnel..." >&2

ARGS="${DDEV_SHARE_ARGS:-}"

echo >&2
echo "Running command: ngrok http $DDEV_LOCAL_URL --log=stdout $ARGS" >&2
echo >&2

URL_FOUND=""
ngrok http "$DDEV_LOCAL_URL" --log=stdout $ARGS 2>&1 | while IFS= read -r line; do
  # Look for the tunnel URL in ngrok's structured log output
  if [[ -z "$URL_FOUND" ]] && [[ "$line" =~ url=(https://[^[:space:]]+) ]]; then
    URL_FOUND="${BASH_REMATCH[1]}"
    echo "$URL_FOUND" # Output to stdout - CRITICAL: This is captured by DDEV
  fi
  # Show non-info output to user (warnings, errors, etc.); show all in verbose mode
  if [[ "${DDEV_VERBOSE:-}" == "true" ]] || [[ ! "$line" =~ " lvl=info " ]]; then
    echo "$line" >&2
  fi
done
