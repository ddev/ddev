#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-websever-dev/php_webserver.bats


@test "check existence/version of various required tools for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    docker exec -t $CONTAINER_NAME php --version | grep "PHP ${PHP_VERSION}"
    docker exec -t $CONTAINER_NAME drush --version
    docker exec -t $CONTAINER_NAME wp --version
    #TODO: Make sure composer cache is used here; mount it?
    docker exec -t $CONTAINER_NAME composer create-project -d //tmp psr/log --no-dev --no-interaction
}

@test "http and https phpstatus access work inside and outside container for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -ssL --fail http://localhost:$HOST_HTTP_PORT/test/phptest.php
    if [ "${OS:-$(uname)}" != "Windows_NT" ] ; then
        # TODO: Why doesn't this work on Windows?
        curl -sSL --fail https://localhost:$HOST_HTTPS_PORT/test/phptest.php
    fi
    docker exec -t $CONTAINER_NAME curl --fail http://localhost/test/phptest.php
    docker exec -t $CONTAINER_NAME curl --fail https://localhost/test/phptest.php
}

@test "enable and disable xdebug for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    docker exec -t $CONTAINER_NAME enable_xdebug
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.remote_enable"
    curl -s localhost:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is enabled"
    docker exec -t $CONTAINER_NAME disable_xdebug
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug does not exist"
    curl -s localhost:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is disabled"
}

@test "verify mailhog for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -s localhost:$HOST_HTTP_PORT/test/test-email.php | grep "Test email sent"
    curl -s --fail localhost:$HOST_HTTP_PORT/test/phptest.php
}

@test "verify PHP ini settings for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    # Default settings for assert.active should be 1
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 1 => 1"
}

@test "verify phpstatus endpoint for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    curl -s localhost:$HOST_HTTP_PORT/phpstatus | egrep "idle processes|php is working"
}

@test "verify error conditions for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    # These are just the standard nginx 403 and 404 pages
    curl localhost:$HOST_HTTP_PORT/ | grep "403 Forbidden"
    curl localhost:$HOST_HTTP_PORT/asdf | grep "404 Not Found"
    # We're just checking the error code here - there's not much more we can do in
    # this case because the container is *NOT* intercepting 50x errors.
    curl -w "%{http_code}" localhost:$HOST_HTTP_PORT/test/500.php | grep 500
    # 400 and 401 errors are intercepted by the same page.
    curl -s -I localhost:$HOST_HTTP_PORT/test/400.php | grep "HTTP/1.1 400"
    curl -s -I localhost:$HOST_HTTP_PORT/test/401.php | grep "HTTP/1.1 401"
}
