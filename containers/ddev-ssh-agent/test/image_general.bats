#!/usr/bin/env bats

# Run these tests from the repo root directory

load functions.sh

function setup {
  basic_setup

  echo "# Starting ${IMAGE}" >&3
  docker run --rm -u "$MOUNTUID:$MOUNTGID" --name=$CONTAINER_NAME -d ${IMAGE}
  containercheck
}

@test "verify apt keys are not expiring" {
  MAX_DAYS_BEFORE_EXPIRATION=90
  if [ "${DDEV_IGNORE_EXPIRING_KEYS:-}" = "true" ]; then
    skip "Skipping because DDEV_IGNORE_EXPIRING_KEYS is set"
  fi
  docker exec -e "max=$MAX_DAYS_BEFORE_EXPIRATION" ${CONTAINER_NAME} bash -c '
    dates=$(apt-key list 2>/dev/null | awk "/\[expires/ { gsub(/[\[\]]/, \"\"); print \$6;}")
    for item in ${dates}; do
      today=$(date -I)
      let diff=($(date +%s -d ${item})-$(date +%s -d ${today}))/86400
      if [ ${diff} -le ${max} ]; then
        exit 1
      fi
    done
  '

}
