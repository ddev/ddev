#!/usr/bin/env bats

# Requires bats-assert and bats-support
# brew tap bats-core/bats-core &&
# brew install bats-core bats-assert bats-support
setup() {
  load setup.sh
}

@test "Verify required binaries are installed in normal image" {
  if [ "${IS_HARDENED}" == "true" ]; then skip "Skipping because IS_HARDENED==true"; fi
  COMMANDS="composer ddev drush8 git magerun magerun2 mkcert mysql mysqladmin mysqldump node npm patch platform ssh sudo symfony terminus wp xdebugctl"
  for item in $COMMANDS; do
    run docker exec "$CONTAINER_NAME" bash -c "command -v $item"
    assert_success
  done
}

@test "Verify some binaries (sudo) are NOT installed in hardened image" {
  if [ "${IS_HARDENED}" != "true" ]; then skip "Skipping because IS_HARDENED==false"; fi
  COMMANDS="sudo terminus"
  for item in $COMMANDS; do
    run docker exec "$CONTAINER_NAME" bash -c "command -v $item"
    assert_failure
  done
}

@test "Verify /var/www/html/vendor/bin is in PATH on ddev/docker exec" {
  run docker exec "$CONTAINER_NAME" bash -c 'echo $PATH'
  assert_success
  assert_output --partial "/var/www/html/vendor/bin"
}

@test "verify that xdebug is disabled by default when using start.sh to start" {
  run docker exec "$CONTAINER_NAME" php --version
  assert_success
  refute_output --partial "with Xdebug"
}

@test "verify that xhprof is disabled by default when using start.sh to start" {
  run docker exec "$CONTAINER_NAME" php --modules
  assert_success
  refute_output --partial "xhprof"
}

@test "verify that composer v2 is installed by default" {
  run docker exec "$CONTAINER_NAME" composer --version
  assert_success
  assert_output --partial "Composer version 2."
}

@test "verify that PHP assertions are enabled by default" {
  run docker exec "$CONTAINER_NAME" php -r 'echo ini_get("zend.assertions");'
  assert_success
  assert_output "1"
}

@test "verify there aren't \"closed keepalive connection\" complaints" {
  run docker logs "$CONTAINER_NAME"
  assert_success
  refute_output --partial "closed keepalive connection"
}

@test "verify access to upstream error messages" {
  run curl -s "127.0.0.1:$HOST_HTTP_PORT/test/upstream-error.php"
  assert_success
  assert_output "Upstream error message"
}

@test "verify that xdebug is not enabled by default" {
  run docker run --rm "$DOCKER_IMAGE" php --version
  assert_success
  refute_output --partial "with Xdebug"
}

@test "verify apt keys are not expiring within ${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90} days" {
  if [ "${DDEV_IGNORE_EXPIRING_KEYS:-}" = "true" ]; then
    skip "Skipping because DDEV_IGNORE_EXPIRING_KEYS is set"
  fi
  run docker cp "${TEST_SCRIPT_DIR}/check_key_expirations.sh" "${CONTAINER_NAME}:/tmp"
  assert_success
  run docker exec -u root -e "DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION=${DDEV_MAX_DAYS_BEFORE_CERT_EXPIRATION:-90}" "${CONTAINER_NAME}" /tmp/check_key_expirations.sh
  assert_success
}

@test "verify python is installed" {
  run docker exec "${CONTAINER_NAME}" python --version
  assert_success
  assert_output --partial "Python 3"
}
