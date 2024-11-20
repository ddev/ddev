#!/usr/bin/env bash

function ddev_custom_init_scripts {
  echo "Loading custom entrypoint config from ${DDEV_WEB_ENTRYPOINT}";
  if ls ${DDEV_WEB_ENTRYPOINT}/*.sh >/dev/null 2>&1; then
    for f in ${DDEV_WEB_ENTRYPOINT}/*.sh; do
      echo "sourcing $f"
      source "$f"
    done
  fi
#      touch "${DDEV_WEB_ENTRYPOINT}/.user_scripts_initialized"
}

