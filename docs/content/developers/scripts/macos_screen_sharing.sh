#!/usr/bin/env bash

# Gives a user every Remote Management privilege on a macOS test runner, so
# Screen Sharing to it works. Remote Management owns port 5900 when it's on
# and rejects a user without Observe and Control as if the password were wrong.
# Turn Remote Management on in System Settings > General > Sharing first:
# kickstart -activate fails over SSH without Full Disk Access.
# Run with sudo. Usage: sudo ./macos_screen_sharing.sh [user] (default testbot)

set -eu -o pipefail

if [ "$(id -u)" != "0" ]; then
  echo "Run this with sudo" >&2 && exit 1
fi

user=${1:-testbot}

/System/Library/CoreServices/RemoteManagement/ARDAgent.app/Contents/Resources/kickstart -configure -access -on -users "${user}" -privs -all -restart -agent

privs=$(dscl . -read "/Users/${user}" naprivs 2>/dev/null | awk '{print $2}')
if [ "${privs}" != "-1073741569" ]; then
  echo "${user} doesn't have every Remote Management privilege (naprivs=${privs:-unset})" >&2 && exit 1
fi

# netstat doesn't list Screen Sharing's port 5900 listener on macOS 27
if /usr/bin/nc -z -G 3 127.0.0.1 5900 >/dev/null 2>&1; then
  echo "${user} has every Remote Management privilege, and port 5900 is listening"
else
  echo "Nothing is listening on port 5900; turn on Remote Management in System Settings > General > Sharing" >&2 && exit 1
fi
