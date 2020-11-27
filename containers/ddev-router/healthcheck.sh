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
configgenerated=false

if nginx -t 2>/dev/null; then
  config=true
  printf "nginx config valid:OK  "
else
  printf "nginx configuration invalid: $(nginx -t 2>&1)"
  exit 2
fi

# If SOMETHING IS WRONG is in default.conf, then the config
# has not been generated yet, or a failure happened generating
if grep -v "SOMETHING IS WRONG" /etc/nginx/conf.d/default.conf >/dev/null; then
  configgenerated=true
  printf "nginx default config:OK "
else
  printf "nginx default.conf not yet generated "
  exit 3
fi

# Check our healthcheck endpoint
if curl -s --fail --connect-timeout 2 --retry 2 http://127.0.0.1/healthcheck; then
  connect=true
  printf "nginx healthcheck endpoint:OK "
else
  printf "healthcheck endpoint not responding "
  exit 4
fi

if [ ${config} = true -a ${connect} = true ]; then
  printf "ddev-router is healthy "
  touch /tmp/healthy
  exit 0
fi

rm -f /tmp/healthy
exit 1
