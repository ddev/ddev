#!/usr/bin/env sh

# This uses /bin/sh, so it doesn't initialize profile/bashrc/etc

# ddev-xhgui healthcheck

set -e

sleeptime=59

# Wait for php webserver to be working
# Since docker doesn't provide a lazy period for startup,
# we track health. If the last check showed healthy
# as determined by existence of /tmp/healthy, then
# sleep at startup. This requires the timeout to be set
# higher than the sleeptime used here.
if [ -f /tmp/healthy ]; then
    printf "container was previously healthy, so sleeping %s seconds before continuing healthcheck... " ${sleeptime}
    sleep ${sleeptime}
fi

if curl --fail -s 127.0.0.1 >/dev/null; then
  phpstatus="true"
  printf "phpstatus:OK "
else
  printf "phpstatus:FAILED "
fi

if [ "${phpstatus}" = "true" ]; then
    touch /tmp/healthy
    exit 0
fi
rm -f /tmp/healthy

exit 1
