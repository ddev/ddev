#!/bin/sh
set -e

profile=/etc/apparmor.d/ddev-gui
if [ -f "$profile" ] && command -v apparmor_parser >/dev/null 2>&1 &&
    [ "$(cat /sys/module/apparmor/parameters/enabled 2>/dev/null)" = "Y" ]; then
    apparmor_parser -r -T -W "$profile" || true
fi
exit 0
