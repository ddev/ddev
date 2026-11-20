#!/bin/bash
# Installs testbot_icinga_push.sh on a macOS testbot as a launchd agent that
# runs every 5 minutes. Safe to rerun: it reloads the agent only when the
# script or the plist changed. testbot_maintenance.sh runs it on every job
# once ~/.config/ddev-icinga/env exists, so runners pick up script changes.

set -eu -o pipefail

LABEL=com.ddev.icinga-push
CONFIG_DIR=${HOME}/.config/ddev-icinga
TARGET=${CONFIG_DIR}/testbot_icinga_push.sh
PLIST=${HOME}/Library/LaunchAgents/${LABEL}.plist
SOURCE=$(cd "$(dirname "$0")" && pwd)/testbot_icinga_push.sh

if [ ! -f "${CONFIG_DIR}/env" ] || [ ! -f "${CONFIG_DIR}/ca.crt" ]; then
  echo "${CONFIG_DIR}/env and ${CONFIG_DIR}/ca.crt must exist first" >&2
  exit 1
fi

changed=false
if ! cmp -s "${SOURCE}" "${TARGET}"; then
  install -m 0755 "${SOURCE}" "${TARGET}"
  changed=true
fi

new_plist=$(cat <<END
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/bash</string>
    <string>${TARGET}</string>
  </array>
  <key>StartInterval</key>
  <integer>300</integer>
  <key>RunAtLoad</key>
  <true/>
  <key>StandardOutPath</key>
  <string>${CONFIG_DIR}/push.log</string>
  <key>StandardErrorPath</key>
  <string>${CONFIG_DIR}/push.log</string>
</dict>
</plist>
END
)
if [ "$(cat "${PLIST}" 2>/dev/null)" != "${new_plist}" ]; then
  mkdir -p "$(dirname "${PLIST}")"
  echo "${new_plist}" >"${PLIST}"
  changed=true
fi

domain=gui/$(id -u)
if [ "${changed}" = "true" ] || ! launchctl print "${domain}/${LABEL}" >/dev/null 2>&1; then
  launchctl bootout "${domain}/${LABEL}" 2>/dev/null || true
  launchctl bootstrap "${domain}" "${PLIST}"
  echo "Loaded ${LABEL} from ${PLIST}"
fi
