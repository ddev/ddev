#!/bin/bash

# ddev-ssh-agent healthcheck

set -eo pipefail
sleeptime=59

# Make sure that both socat and ssh-agent are running
# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping ${sleeptime} seconds before continuing healthcheck...  "
    sleep ${sleeptime}
fi
if killall -0 socat ssh-agent; then
    printf "healthy"
    touch /tmp/healthy
    exit 0
fi
rm -f /tmp/healthy
exit 1

