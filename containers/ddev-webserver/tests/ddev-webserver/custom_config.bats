#!/usr/bin/env bats

@test "verify customization of php configuration" {
    # Make sure the custom php configuration in ddev_config/php is activated.
    # Verify that /mnt/ddev_config is mounted and we have php overrides there.
    # First see if /mnt/ddev_config "works" for php
    docker exec -t $CONTAINER_NAME bash -c "ls -l /mnt/ddev_config/php/my-php.ini || (echo 'Failed to ls /mnt/ddev_config' && exit 201)"


    # With overridden value we should have assert.active=0, not the default
    echo "--- Check that assert.active override is working"
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 0 => 0" >/dev/null

    # Make sure that our nginx override providing /junker99 works correctly
    curl -s http://127.0.0.1:$HOST_HTTP_PORT/junker99 | grep 'junker99!'
    #TODO: Why wouldn't this work on Windows?
    if [ "${OS:-$(uname)}" != "Windows_NT" ]; then
        curl -s https://127.0.0.1:$HOST_HTTPS_PORT/junker99 | grep 'junker99!'
    fi
}
