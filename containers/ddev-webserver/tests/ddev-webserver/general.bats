#!/usr/bin/env bats

@test "Verify required binaries are installed in container" {
    COMMANDS="ddev-live terminus drupal magerun magerun2 drush mkcert composer"
    for item in $COMMANDS; do
       docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null"
    done
}
