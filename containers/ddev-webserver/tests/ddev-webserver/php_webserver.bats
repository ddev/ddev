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
    if [ "${PHP_VERSION}" = "8.3" ]; then skip "Skipping for PHP_VERSION=8.3 because no xdebug yet"; fi

    CURRENT_ARCH=$(../get_arch.sh)

    docker exec -t $CONTAINER_NAME enable_xdebug
    if [[ ${PHP_VERSION} != 8.? ]] ; then
      docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.remote_enable"
    else
      docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.mode"
    fi
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is enabled"
    docker exec -t $CONTAINER_NAME disable_xdebug
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.*does not exist"
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php | grep "Xdebug is disabled"
}

@test "enable and disable xhprof for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
    if [ "${PHP_VERSION}" = "8.3" ]; then skip "Skipping for PHP_VERSION=8.3 because no xhprof yet"; fi

    CURRENT_ARCH=$(../get_arch.sh)

    docker exec -t $CONTAINER_NAME enable_xhprof
    docker exec -t $CONTAINER_NAME php --re xhprof | grep "xhprof.output_dir"
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php | grep "XHProf is enabled"
    docker exec -t $CONTAINER_NAME disable_xhprof
    docker exec -t $CONTAINER_NAME php --re xhprof | grep "does not exist"
    curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php | grep "XHProf is disabled"
}

@test "verify mailhog for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  if [ ${IS_HARDENED} == "true" ]; then skip "Skipping because mailhog is not installed on hardened prod image"; fi
  curl -s 127.0.0.1:$HOST_HTTP_PORT/test/test-email.php | grep "Test email sent"
  curl -s --fail 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
}

@test "verify PHP ini settings for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # Default settings for assert.active should be 1
  if [[ ${PHP_VERSION} != 8.? ]]; then
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 1 => 1"
  else
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> On => On"
  fi
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

@test "verify key php extensions are loaded on PHP${PHP_VERSION}" {
  if [ "${WEBSERVER_TYPE}" = "apache-fpm" ]; then skip "Skipping on apache-fpm because we don't have to do this twice"; fi

  extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring memcached mysqli pgsql readline redis soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
  case ${PHP_VERSION} in
  5.6)
    extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring mysql pgsql readline soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
    ;;
  7.[01234])
    extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring mysqli pgsql readline soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
    ;;
  8.0)
    extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring memcached mysqli pgsql readline redis soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
    ;;
  8.1)
    extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring memcached mysqli pgsql readline redis soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
    ;;
  8.2)
    extensions="apcu bcmath bz2 curl gd imagick intl json ldap mbstring memcached mysqli pgsql readline redis soap sqlite3 uploadprogress xhprof xml xmlrpc zip"
    ;;
  8.3)
    # TODO: Update when more extensions are available for PHP 8.3
    extensions="bcmath bz2 curl gd intl ldap mbstring mysqli pgsql readline soap sqlite3 xml zip"
    ;;
  esac

  # TODO: Remove the if block when xdebug and xhprof are available for PHP 8.3
  if [ "${PHP_VERSION}" != "8.3" ]; then
    run docker exec -t $CONTAINER_NAME enable_xdebug
    run docker exec -t $CONTAINER_NAME enable_xhprof
  fi

  run docker exec -t $CONTAINER_NAME bash -c "php -r \"print_r(get_loaded_extensions());\" 2>/dev/null | tr -d '\r\n'"
  loaded="${output}"
  # echo "# loaded=${output}" >&3
  for item in $extensions; do
#    echo "# extension: $item on PHP${PHP_VERSION}" >&3
    grep -q "=> $item " <<< ${loaded} || (echo "# extension ${item} not loaded" >&3 && false)
  done

  # TODO: Remove the if block when xdebug and xhprof are available for PHP 8.3
  if [ "${PHP_VERSION}" != "8.3" ]; then
    run docker exec -t $CONTAINER_NAME disable_xdebug
    run docker exec -t $CONTAINER_NAME disable_xhprof
  fi

}

@test "verify htaccess doesn't break ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  docker cp tests/ddev-webserver/testdata/nginx/auth.conf ${CONTAINER_NAME}:/etc/nginx/common.d
  docker cp tests/ddev-webserver/testdata/nginx/junkpass ${CONTAINER_NAME}:/tmp
  docker cp tests/ddev-webserver/testdata/apache/auth.conf ${CONTAINER_NAME}:/etc/apache2/conf-enabled
  # Reload webserver
  if [ "${WEBSERVER_TYPE}" = "apache-fpm" ]; then
    docker exec ${CONTAINER_NAME} apache2ctl -k graceful
  else
    docker exec ${CONTAINER_NAME} nginx -s reload
  fi
  sleep 2
  # Make sure we can hit /phpstatus without auth
  run curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$HOST_HTTP_PORT/phpstatus
  echo "# phpstatus status=$output"
  [ "$status" = 0 ]
  [ "$output" = "200" ]
  curl --fail -s http://127.0.0.1:$HOST_HTTP_PORT/phpstatus | egrep "idle processes|php is working"
  # Make sure the auth requirement is actually working
  curlstmt="curl --fail -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$HOST_HTTP_PORT/test/phptest.php"
  run ${curlstmt}
  [ "$output" = "401" ]

  # Make sure it works with auth when hitting phptest.php
  AUTH=$(echo -ne "junk:junk" | base64)
  curl --fail --header "Authorization: Basic $AUTH" 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  docker exec ${CONTAINER_NAME} rm /etc/nginx/common.d/auth.conf /etc/apache2/conf-enabled/auth.conf
  docker exec ${CONTAINER_NAME} kill -HUP 1
}
