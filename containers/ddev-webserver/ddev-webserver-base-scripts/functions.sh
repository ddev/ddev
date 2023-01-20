#!/bin/bash

function ddev_custom_init_scripts {
  if [[ -n $(find ${ENTRYPOINT} -type f -regex ".*\.\(sh\)") ]] && [[ ! -f "${ENTRYPOINT}/.user_scripts_initialized" ]] ; then
      echo "Loading custom entrypoint config from ${ENTRYPOINT}";
      for f in ${ENTRYPOINT}/*.sh; do
        echo "sourcing web-entrypoint.d/$f"
        . "$f"
      done
#      touch "${ENTRYPOINT}/.user_scripts_initialized"
  fi
}
