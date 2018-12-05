#!/bin/bash

# bgsync healthcheck

set -eo pipefail
set -o nounset
set -o errexit


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
  echo "Sync is not working after $WAIT_FOR_SYNC seconds"
  exit 101
fi
echo "Sync successful for $CHECKFILE"