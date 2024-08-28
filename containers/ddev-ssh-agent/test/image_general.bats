#!/usr/bin/env bats

# Run these tests from the repo root directory

load functions.sh

function setup {
  basic_setup

  echo "# Starting ${IMAGE}" >&3
  docker run --rm -u "$MOUNTUID:$MOUNTGID" --name=$CONTAINER_NAME -d ${IMAGE}
  containercheck
}

@test "verify apt keys are not expiring within ${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90} days" {
  if [ "${DDEV_IGNORE_EXPIRING_KEYS:-}" = "true" ]; then
    skip "Skipping because DDEV_IGNORE_EXPIRING_KEYS is set"
  fi
  docker cp ${TEST_SCRIPT_DIR}/check_key_expirations.sh ${CONTAINER_NAME}:/tmp
  docker exec -u root -e "DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION=$DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION" ${CONTAINER_NAME} /tmp/check_key_expirations.sh

}
