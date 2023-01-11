#!/bin/bash

function ddev_custom_init_scripts {
  if [[ -n $(find ${ENTRYPOINT} -type f -regex ".*\.\(sh\)") ]] && [[ ! -f "${ENTRYPOINT}/.user_scripts_initialized" ]] ; then
      echo "Loading custom entrypoint config from ${ENTRYPOINT}";
      for f in ${ENTRYPOINT}/*.sh; do
          echo "Executing $f"
          if [[ -x "$f" ]]; then
              if ! "$f"; then
                  echo "Failed executing $f"
                  return 1
              fi
          else
            echo "Sourcing $f as it is not executable by the current user, any error may cause initialization to fail"
            . "$f"
          fi
      done
#      touch "${ENTRYPOINT}/.user_scripts_initialized"
  fi
}
