#!/usr/bin/env bats

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

@test "Verify PATH contains all expected directories in non-interactive, interactive, and login shells" {
  local check='
    for p in \
      "$HOME/.local/bin" \
      "$HOME/bin" \
      "${DDEV_COMPOSER_ROOT:-/var/www/html}/vendor/bin" \
      "/var/www/html/bin" \
      "/mnt/ddev-global-cache/global-commands/web"; do
      case ":$PATH:" in
        *":$p:"*) ;;
        *) echo "Missing from PATH: $p" >&2; exit 1 ;;
      esac
    done
  '

  run docker exec "$CONTAINER_NAME" bash -c "$check"
  assert_success

  run docker exec -t "$CONTAINER_NAME" bash -ic "$check"
  assert_success

  run docker exec "$CONTAINER_NAME" bash -lc "$check"
  assert_success
}

@test "Verify PATH is identical across non-interactive, interactive, and login shells" {
  run docker exec "$CONTAINER_NAME" bash -c 'echo $PATH'
  assert_success
  local noninteractive_path="$output"

  run docker exec -t "$CONTAINER_NAME" bash -ic 'echo $PATH'
  assert_success
  # -t allocates a pseudo-TTY which adds \r to output; strip it before comparing
  assert_equal "${output//$'\r'/}" "$noninteractive_path"

  run docker exec "$CONTAINER_NAME" bash -lc 'echo $PATH'
  assert_success
  assert_equal "${output}" "$noninteractive_path"
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

@test "verify mariadb compat wrappers are installed and produce no deprecation warnings" {
  for cmd in mysql mysqld mysqldump mysqladmin mysqlcheck; do
    run docker exec "$CONTAINER_NAME" bash -c "command -v $cmd"
    assert_success
  done
  run docker exec "$CONTAINER_NAME" bash -c "mysql --version 2>&1"
  assert_success
  refute_output --partial "Deprecated program name"
}

@test "verify mariadb compat wrappers are removed for MySQL and reinstalled for MariaDB" {
  # Remove wrappers by switching to MySQL
  run docker exec -e DDEV_DATABASE="mysql:8.0" "$CONTAINER_NAME" mariadb-compat-install.sh
  assert_success
  run docker exec "$CONTAINER_NAME" bash -c "[ -f /usr/local/bin/mysql ] && head -n 3 /usr/local/bin/mysql | grep -q '#ddev-generated' && echo 'wrapper exists' || echo 'wrapper absent'"
  assert_output "wrapper absent"
  # Reinstall wrappers by switching back to MariaDB
  run docker exec -e DDEV_DATABASE="mariadb:11.8" "$CONTAINER_NAME" mariadb-compat-install.sh
  assert_success
  run docker exec "$CONTAINER_NAME" bash -c "command -v mysql"
  assert_success
}

@test "verify mariadb skip-ssl wrappers are installed and contain --skip-ssl-verify-server-cert" {
  for cmd in mariadb mariadb-admin mariadb-dump; do
    run docker exec "$CONTAINER_NAME" bash -c "command -v $cmd"
    assert_success
    run docker exec "$CONTAINER_NAME" bash -c "grep -q -- '--skip-ssl-verify-server-cert' /usr/local/bin/$cmd"
    assert_success
  done
}

@test "verify mariadb skip-ssl wrappers are removed for MySQL and reinstalled for MariaDB" {
  # Remove wrappers by switching to MySQL
  run docker exec -e DDEV_DATABASE="mysql:8.0" "$CONTAINER_NAME" mariadb-skip-ssl-wrapper-install.sh
  assert_success
  run docker exec "$CONTAINER_NAME" bash -c "[ -f /usr/local/bin/mariadb ] && head -n 3 /usr/local/bin/mariadb | grep -q '#ddev-generated' && echo 'wrapper exists' || echo 'wrapper absent'"
  assert_output "wrapper absent"
  # Reinstall wrappers by switching back to MariaDB
  run docker exec -e DDEV_DATABASE="mariadb:11.8" "$CONTAINER_NAME" mariadb-skip-ssl-wrapper-install.sh
  assert_success
  run docker exec "$CONTAINER_NAME" bash -c "grep -q -- '--skip-ssl-verify-server-cert' /usr/local/bin/mariadb"
  assert_success
}

@test "verify python is installed" {
  run docker exec "${CONTAINER_NAME}" python --version
  assert_success
  assert_output --partial "Python 3"
}
