#!/usr/bin/env bats

@test "Verify required binaries are installed in container" {
    COMMANDS="composer ddev-live drupal drush magerun magerun2 mkcert terminus wp"
    for item in $COMMANDS; do
       docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null"
    done
}
