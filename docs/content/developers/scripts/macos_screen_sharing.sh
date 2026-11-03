#!/usr/bin/env bash

# Turns on Screen Sharing on a macOS test runner and turns off Remote
# Management, which otherwise takes over port 5900 and rejects users without
# Remote Management privileges as if their password were wrong.
# Run with sudo.

set -eu -o pipefail

if [ "$(id -u)" != "0" ]; then
  echo "Run this with sudo" >&2 && exit 1
fi

/System/Library/CoreServices/RemoteManagement/ARDAgent.app/Contents/Resources/kickstart -deactivate -configure -access -off
launchctl enable system/com.apple.screensharing
# bootstrap fails when the job is already loaded, which is fine
launchctl bootstrap system /System/Library/LaunchDaemons/com.apple.screensharing.plist 2>/dev/null || true

if netstat -an -p tcp | grep -qE '\.5900 .*LISTEN'; then
  echo "Screen Sharing is listening on port 5900"
else
  echo "Nothing is listening on port 5900; turn on Screen Sharing in System Settings > General > Sharing" >&2 && exit 1
fi
