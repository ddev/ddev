#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-webserver/php_webserver.bats

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
  CURRENT_ARCH=$(../get_arch.sh)

  docker exec -t $CONTAINER_NAME enable_xhprof
  docker exec -t $CONTAINER_NAME php --re xhprof | grep "xhprof.output_dir"
  curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php | grep "XHProf is enabled"
  docker exec -t $CONTAINER_NAME disable_xhprof
  docker exec -t $CONTAINER_NAME php --re xhprof | grep "does not exist"
  curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php | grep "XHProf is disabled"
}

@test "verify mailpit for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  if [ ${IS_HARDENED} == "true" ]; then skip "Skipping because mailpit is not installed on hardened prod image"; fi
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

  # Conditional extension list based on Debian Trixie Sury repository availability
  # Base extensions that should always be available
  extensions="apcu bcmath bz2 curl gd imagick intl ldap mbstring mysqli pgsql readline soap sqlite3 uploadprogress xhprof xml xmlrpc zip"

  # Conditionally add extensions based on PHP version and known Sury repository issues
  # https://codeberg.org/oerdnj/deb.sury.org/issues
  case ${PHP_VERSION} in
  5.6)
    extensions="$extensions json memcached redis xdebug"
    ;;
  7.0|7.1|7.2|7.3)
    extensions="$extensions json memcached xdebug"
    # php7.0-7.3: redis arm64 is missing in Debian Trixie Sury
    if [ "$(uname -m)" != "aarch64" ] && [ "$(uname -m)" != "arm64" ]; then
      extensions="$extensions redis"
    fi
    ;;
  7.4)
    extensions="$extensions json memcached redis xdebug"
    ;;
  8.0|8.1|8.2|8.3|8.4)
    extensions="$extensions memcached redis xdebug"
    ;;
  8.5)
    # TODO: php8.5: memcached not yet available in Debian Trixie Sury arm64
    extensions="$extensions redis xdebug"
    ;;
  *)
    # Default fallback for future PHP versions - assume redis available
    extensions="$extensions redis"
    ;;
  esac

  # Load xhprof first, then xdebug, because loading xhprof disables xdebug
  run docker exec $CONTAINER_NAME enable_xhprof
  run docker exec $CONTAINER_NAME enable_xdebug
  run docker exec $CONTAINER_NAME bash -c "php -r 'foreach (get_loaded_extensions() as \$e) echo \$e, PHP_EOL;' 2>/dev/null"
  loaded="${output}"
  # echo "# loaded=${output}" >&3
  for item in $extensions; do
    # echo "# extension: $item on php${PHP_VERSION}" >&3
    grep -qx "$item" <<< "${loaded}" || (echo "# extension ${item} not loaded" >&3 && false)
  done

  run docker exec $CONTAINER_NAME disable_xhprof
  run docker exec $CONTAINER_NAME disable_xdebug
}

@test "verify that both nginx logs and fpm logs are being tailed (${WEBSERVER_TYPE})" {
  curl -sSL http://127.0.0.1:$HOST_HTTP_PORT/test/fatal.php >/dev/null 2>&1
  # php-fpm message direct
  docker logs ${CONTAINER_NAME} 2>&1 | grep "PHP Fatal error:  Fatal error in"
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
  docker exec ${CONTAINER_NAME} kill -USR2 1
}

