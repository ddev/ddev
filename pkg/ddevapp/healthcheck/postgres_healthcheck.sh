#!/bin/bash

## postgres healthcheck
##dev-generated

set -eo pipefail
sleeptime=59

# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping ${sleeptime} seconds before continuing healthcheck...  "
    sleep ${sleeptime}
fi

# If we can now access the server, we're healthy and ready
if  pg_isready >/dev/null;  then
    printf "pg_isready: healthy"
    touch /tmp/healthy
    exit 0
fi

rm -f /tmp/healthy
exit 1

