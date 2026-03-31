#!/usr/bin/env bats

setup() {
  load setup.sh
}

@test "verify customization of php configuration" {
  # Make sure the custom php configuration in ddev_config/php is activated.
  # Verify that /mnt/ddev_config is mounted and we have php overrides there.
  run docker exec -t $CONTAINER_NAME bash -c "ls -l /mnt/ddev_config/php/my-php.ini"
  assert_success

  # With overridden value we should have assert.active=0, not the default
  run docker exec -t $CONTAINER_NAME php -i
  assert_success
  assert_output --regexp "assert.active.*=> Off => Off"

  # Make sure that our nginx override providing /junker99 works correctly
  run curl -s "http://127.0.0.1:$HOST_HTTP_PORT/junker99"
  assert_success
  assert_output --partial "junker99!"
  #TODO: Why wouldn't this work on Windows?
  if [ "${OS:-$(uname)}" != "Windows_NT" ]; then
    run curl -s "https://127.0.0.1:$HOST_HTTPS_PORT/junker99"
    assert_success
    assert_output --partial "junker99!"
  fi
}
