#!/bin/bash

## traefik health check
set -eu -o pipefail
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

# If /tmp/initializing, it means we're loading the default starter database
if [ -f /tmp/initializing ]; then
  printf "initializing"
  exit 1
fi

# If we can now access the traefik ping endpoint, then we're healthy
# We should be able to use `traefik healthcheck --ping` but it doesn't work if
# using nonstandard port (always tries port 8080 even if traefik port is something else)
if curl -s -f http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/ping ; then
    printf "healthy"
    touch /tmp/healthy
    exit 0
fi

rm -f /tmp/healthy
exit 1

