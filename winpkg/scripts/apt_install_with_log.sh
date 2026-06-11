#!/bin/bash
# Run `apt-get install -y` for the given packages, capturing full output
# to a log file. On stdout, print a concise diagnostic (status + error
# lines + tail) so the NSIS installer's small (~1024 byte) capture
# buffer can show the actual cause of failure. Exits with apt-get's
# exit code.
#
# Acquire::Retries=5 retries each fetch up to 5 times within a single
# apt-get invocation. Empirically required for upstream mirrors that
# intermittently return 502 (e.g. Parrot's deb.parrot.sh CDN) — without
# it, a single transient HTTP failure aborts the whole install.
#
# Usage: apt_install_with_log.sh <stage_name> <package> [package ...]
#   stage_name: short tag used in the log file path, e.g. "essential",
#               "docker", "ddev". Each stage gets its own log file so
#               later failures don't overwrite earlier ones.
#
# Optional env: APT_EXTRA_ARGS — additional apt-get args, word-split.
#   Example: APT_EXTRA_ARGS="--no-install-recommends"

set -u

STAGE="${1:-unknown}"
shift || true

LOGFILE="/tmp/ddev_apt_${STAGE}.log"

# shellcheck disable=SC2086
DEBIAN_FRONTEND=noninteractive apt-get \
    -o Acquire::Retries=5 \
    install -y ${APT_EXTRA_ARGS:-} "$@" >"$LOGFILE" 2>&1
rc=$?

# NSIS's nsExec::ExecToStack buffer is ~1024 bytes, so be brief.
# Prefer error lines (Err:/E:/Error:/W: Failed) since those name the cause;
# fall back to the last 12 lines if no error lines were emitted.
echo "exit=$rc stage=$STAGE log=$LOGFILE"
ERR_LINES=$(grep -E '^(Err|E:|Error|W: Failed)' "$LOGFILE" 2>/dev/null | head -10)
if [ -n "$ERR_LINES" ]; then
    echo "--- error lines ---"
    printf '%s\n' "$ERR_LINES"
else
    echo "--- last 12 lines ---"
    tail -12 "$LOGFILE"
fi

exit "$rc"
