#!/bin/bash

echo "Loading custom entrypoint config from '${DDEV_WEB_ENTRYPOINT}'";
for f in ${DDEV_WEB_ENTRYPOINT}/*.sh; do
  echo "sourcing web-entrypoint.d/$f"
  . "$f"
done
#      touch "${DDEV_WEB_ENTRYPOINT}/.user_scripts_initialized"
