#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-websever-dev/php_webserver.bats

@test "http and https phpstatus access work inside and outside container for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -sSL --fail http://127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
    if [ "${OS:-$(uname)}" != "Windows_NT" ] ; then
        # TODO: Why doesn't this work on Windows?
        curl -sSL --fail https://127.0.0.1:$HOST_HTTPS_PORT/test/phptest.php
    fi
    docker exec -t $CONTAINER_NAME curl --fail http://127.0.0.1/test/phptest.php
    docker exec -t $CONTAINER_NAME curl --fail https://127.0.0.1/test/phptest.php
}

@test "enable and disable xdebug for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    docker exec -t $CONTAINER_NAME enable_xdebug
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.remote_enable"
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is enabled"
    docker exec -t $CONTAINER_NAME disable_xdebug
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug does not exist"
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is disabled"
}

@test "verify mailhog for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/test-email.php | grep "Test email sent"
    curl -s --fail 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
}

@test "verify PHP ini settings for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    # Default settings for assert.active should be 1
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 1 => 1"
}

@test "verify phpstatus endpoint for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -s 127.0.0.1:$HOST_HTTP_PORT/phpstatus | egrep "idle processes|php is working"
}

@test "verify error conditions for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    # These are just the standard nginx 403 and 404 pages
    curl 127.0.0.1:$HOST_HTTP_PORT/asdf | grep "404 Not Found"
    # We're just checking the error code here - there's not much more we can do in
    # this case because the container is *NOT* intercepting 50x errors.
    for item in 400 401 500; do
        curl -w "%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/${item}.php | grep $item
    done
}

@test "verify that test/phptest.php is interpreted ($project_type)" {
	curl --fail 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
}
