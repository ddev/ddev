#!/usr/bin/env bash

# This script monitors stderr output from a long-running process (like traefik)
# and logs only ERR (error) messages to /tmp/ddev-traefik-errors.txt.
# WRN (warning) and debug lines are not written there so they do not appear
# as "router configuration problems" in ddev start / ddev list.

# Usage: monitor-traefik-stderr.sh command [args...]
# Example: monitor-traefik-stderr.sh traefik --configFile=/config.yaml

if [ "$#" -lt 1 ]; then
  echo "Usage: $(basename "$0") command [args...]"
  exit 1
fi

error_file="/tmp/ddev-traefik-errors.txt"

# Run the command in background, capture output, filter for ERR only, and log
"$@" 2>&1 | while IFS= read -r line; do
  echo "$line"
  if echo "$line" | grep -qE '\bERR\b'; then
    # Strip timestamp (everything after first space) and only add if not duplicate
    msg=$(echo "$line" | cut -d' ' -f2-)
    if ! grep -qF "$msg" "${error_file}" 2>/dev/null; then
      echo "$msg" >> "${error_file}"
    fi
  fi
done
