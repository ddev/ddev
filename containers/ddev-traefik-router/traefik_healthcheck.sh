#!/usr/bin/env bash

## traefik health check
set -u -o pipefail
sleeptime=59

# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping %s seconds before continuing healthcheck...  " ${sleeptime}
    sleep ${sleeptime}
fi

# If we can now access the traefik ping endpoint, then we're healthy
# Technique from https://doc.traefik.io/traefik/operations/ping/#entrypoint
check=$(traefik healthcheck --ping.entryPoint=ping --entryPoints.ping.address=127.0.0.1:${TRAEFIK_MONITOR_PORT} --ping 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
    printf "%s" "${check}"
    touch /tmp/healthy
    exit 0
fi

printf "Traefik healthcheck failed: %s" "${check}"
rm -f /tmp/healthy
exit $exit_code
