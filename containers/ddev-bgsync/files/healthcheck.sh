#!/bin/bash

# bgsync healthcheck

set -eo pipefail
set -o nounset
set -o errexit

WAIT_FOR_SYNC=2
CHECKFILE="healthcheck.$(date +%s)"

SYNC_SOURCE=${SYNC_SOURCE:-/source}
SYNC_DESTINATION=${SYNC_DESTINATION:-/destination}
HEALTHCHECK_DIR=.bgsync_healthcheck

mkdir -p $SYNC_DESTINATION/$HEALTHCHECK_DIR $SYNC_DESTINATION/$HEALTHCHECK_DIR
touch ${SYNC_SOURCE}/${HEALTHCHECK_DIR}/$CHECKFILE && sleep $WAIT_FOR_SYNC
if [ ! -f ${SYNC_DESTINATION}/${HEALTHCHECK_DIR}/$CHECKFILE ]; then
  echo "Sync is not working after $WAIT_FOR_SYNC seconds"
  exit 101
fi
echo "Sync successful for $CHECKFILE"
rm ${SYNC_SOURCE}/${HEALTHCHECK_DIR}/$CHECKFILE ${SYNC_DESTINATION}/${HEALTHCHECK_DIR}/$CHECKFILE