#!/bin/bash
# Ensure /etc/wsl.conf has [boot] systemd=true. Preserves all other
# settings. Prints a short status line so the installer log captures what
# happened. Exit codes:
#   0  systemd is already enabled (no action needed)
#   1  systemd was just enabled (caller MUST wsl --terminate this distro
#      and re-enter it so the change takes effect)
#   2  error (could not read/write /etc/wsl.conf)
#
# The systemd line is normalized to exactly `systemd=true` (no surrounding
# whitespace) so future runs can detect it idempotently.

set -u

CONF=/etc/wsl.conf

# If systemd is already running as PID 1 (WSL enables it by default on modern
# Ubuntu/Debian), no restart is needed regardless of whether the config file
# explicitly says so. We still write the config for idempotency, but exit 0.
SYSTEMD_RUNNING=false
if [ "$(ps -p 1 -o comm= 2>/dev/null)" = "systemd" ]; then
    SYSTEMD_RUNNING=true
fi

if ! touch "$CONF" 2>/dev/null; then
    echo "ERROR: cannot write $CONF"
    exit 2
fi

# Case 1: no [boot] section at all — append one.
if ! grep -q '^\[boot\]' "$CONF"; then
    {
        echo ""
        echo "[boot]"
        echo "systemd=true"
    } >> "$CONF"
    echo "STATUS: added [boot] section with systemd=true to $CONF"
    [ "$SYSTEMD_RUNNING" = "true" ] && { echo "STATUS: systemd already running as PID 1; no restart needed"; exit 0; }
    exit 1
fi

# Case 2: [boot] section exists. Check current systemd= value, strictly
# within the [boot] section (other sections may have unrelated keys).
CURRENT=$(awk '
    /^\[boot\]/   { in_boot=1; next }
    /^\[/         { in_boot=0; next }
    in_boot && /^[[:space:]]*systemd[[:space:]]*=/ { print; exit }
' "$CONF")

if [ -z "$CURRENT" ]; then
    # [boot] exists but no systemd line. Insert one right after [boot].
    awk '
        /^\[boot\]/ && !done { print; print "systemd=true"; done=1; next }
        { print }
    ' "$CONF" > "${CONF}.new" && mv "${CONF}.new" "$CONF"
    echo "STATUS: added systemd=true to existing [boot] section in $CONF"
    [ "$SYSTEMD_RUNNING" = "true" ] && { echo "STATUS: systemd already running as PID 1; no restart needed"; exit 0; }
    exit 1
fi

# Normalize the captured value: strip whitespace, lowercase the right side.
VALUE=$(echo "$CURRENT" | sed 's/[[:space:]]//g' | cut -d= -f2- | tr '[:upper:]' '[:lower:]')
if [ "$VALUE" = "true" ]; then
    echo "STATUS: systemd already enabled in $CONF"
    exit 0
fi

# Case 3: systemd= is set to something other than true (false, 0, etc.).
# Rewrite that one line within [boot], leave the rest of the file alone.
awk '
    /^\[boot\]/   { in_boot=1; print; next }
    /^\[/         { in_boot=0; print; next }
    in_boot && /^[[:space:]]*systemd[[:space:]]*=/ { print "systemd=true"; next }
    { print }
' "$CONF" > "${CONF}.new" && mv "${CONF}.new" "$CONF"
echo "STATUS: changed systemd setting to true in $CONF (was: $CURRENT)"
[ "$SYSTEMD_RUNNING" = "true" ] && { echo "STATUS: systemd already running as PID 1; no restart needed"; exit 0; }
exit 1
