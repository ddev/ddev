#!/bin/bash
set -eu
set -o pipefail

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

config=false
connect=false

if nginx -t ; then
    config=true
    printf "nginx config: OK  "
else
  printf "nginx configuration invalid: $(nginx -t)"
  exit 2
fi

# Check our healthcheck endpoint
if curl --fail --connect-timeout 2 --retry 2 http://127.0.0.1/healthcheck; then
    connect=true
    echo "nginx healthcheck endpoint: OK "
else
    echo "ddev-router healthcheck endpoint not responding "
fi

if [ ${config} = true -a ${connect} = true ]; then
    printf "healthy"
    touch /tmp/healthy
    exit 0
fi

rm -f /tmp/healthy
exit 1
