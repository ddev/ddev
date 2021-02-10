#!/bin/bash

## mysql health check for docker

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

# If /tmp/initializing, it means we're loading the default starter database
if [ -f /tmp/initializing ]; then
  printf "initializing"
  exit 1
fi

# If mariabackup or xtrabackup is running (and not initializing)
# It means snapshot restore is in progress
if killall -0 mariabackup || killall -0 xtrabackup ; then
  printf "restoring snapshot"
  touch /tmp/healthy
  exit 0
fi

# If we can now access the server, we're healthy and ready
if  mysql --host=127.0.0.1 -udb -pdb --database=db -e "SHOW DATABASES LIKE 'db';" >/dev/null;  then
    printf "healthy"
    touch /tmp/healthy
    exit 0
fi

rm -f /tmp/healthy
exit 1

