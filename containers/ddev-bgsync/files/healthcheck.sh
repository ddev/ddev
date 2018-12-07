#!/bin/bash

# bgsync healthcheck

set -eo pipefail
set -o nounset
set -o errexit



if [ ! -f "/var/tmp/unison_start_authorized" ] ; then
  echo -n "unison start has not yet been authorized"
  exit 101
fi

if ! pkill -0 unison ; then
  echo -n "unison does not appear to be running"
  exit 102
fi

WAIT_FOR_SYNC=2
CHECKFILE="healthcheck.$(date +%Y%m%d%H%M%S)"

SYNC_SOURCE=${SYNC_SOURCE:-/source}
SYNC_DESTINATION=${SYNC_DESTINATION:-/destination}
HEALTHCHECK_DIR=.bgsync_healthcheck

function cleanup {
    rm -f ${SYNC_SOURCE}/${HEALTHCHECK_DIR}/healthcheck.* ${SYNC_DESTINATION}/${HEALTHCHECK_DIR}/healthcheck.*
}
trap cleanup EXIT

mkdir -p "$SYNC_DESTINATION/$HEALTHCHECK_DIR" "$SYNC_SOURCE/$HEALTHCHECK_DIR"
touch "${SYNC_SOURCE}/${HEALTHCHECK_DIR}/$CHECKFILE" && sleep "$WAIT_FOR_SYNC"
if [ ! -f "${SYNC_DESTINATION}/${HEALTHCHECK_DIR}/$CHECKFILE" ]; then
  echo -n "Sync not yet fully active"
else
    echo -n "Sync active"
    touch /var/tmp/sync_active.txt
fi

exit 0
