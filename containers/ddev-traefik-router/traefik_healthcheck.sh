#!/usr/bin/env bash

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

# If we can now access the traefik ping endpoint, then we're healthy
if traefik healthcheck --ping --configFile=/mnt/ddev-global-cache/traefik/.static_config.yaml; then
    touch /tmp/healthy
    exit 0
fi

rm -f /tmp/healthy
exit 1
