#!/usr/bin/env bash

# PID 1 for the web container. /start.sh is run later via docker exec,
# and it and supervisor log to /proc/1/fd/1 to reach docker logs.

set -x
set -eu -o pipefail

# Kill process 1 + process group if this exist or fails
trap "trap - SIGTERM && kill -- -1" SIGINT SIGTERM EXIT SIGHUP SIGQUIT

# Run sleep in background so bash can process signals during `wait`;
# a foreground child would defer SIGTERM until the full stop_grace_period.
sleep infinity &
wait
