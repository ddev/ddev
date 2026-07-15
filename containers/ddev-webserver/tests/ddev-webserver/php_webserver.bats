#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-webserver/php_webserver.bats

setup() {
  load setup.sh
}

@test "HTTP_HOST passed to PHP preserves nonstandard port for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # Debian's nginx-common overrides HTTP_HOST with port-stripped $host in
  # /etc/nginx/fastcgi_params (Debian bug #1126960 security workaround),
  # which breaks app-generated absolute URLs when router_http(s)_port
  # is not 80/443. Make sure the client's Host header, including any
  # nonstandard port, reaches PHP unchanged.
  run curl -s --fail -H "Host: hosttest.ddev.site:8443" http://127.0.0.1:$HOST_HTTP_PORT/test/hosttest.php
  assert_success
  assert_output --partial "HTTP_HOST=hosttest.ddev.site:8443"
  run curl -sk --fail -H "Host: hosttest.ddev.site:8443" https://127.0.0.1:$HOST_HTTPS_PORT/test/hosttest.php
  assert_success
  assert_output --partial "HTTP_HOST=hosttest.ddev.site:8443"
}

@test "http and https phpstatus access work inside and outside container for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run curl -sSL --fail http://127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  assert_success
  if [ "${OS:-$(uname)}" != "Windows_NT" ] ; then
    # TODO: Why doesn't this work on Windows?
    run curl -sSL --fail https://127.0.0.1:$HOST_HTTPS_PORT/test/phptest.php
    assert_success
  fi
  run docker exec -t $CONTAINER_NAME curl --fail http://127.0.0.1/test/phptest.php
  assert_success
  run docker exec -t $CONTAINER_NAME curl --fail https://127.0.0.1/test/phptest.php
  assert_success
}

@test "update-alternatives can switch PHP without world-writable alternatives directories for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker exec -t "$CONTAINER_NAME" bash -c 'test "$(stat -c %A /etc/alternatives | cut -c9)" != "w" && test "$(stat -c %A /var/lib/dpkg/alternatives | cut -c9)" != "w"'
  assert_success
  run docker exec -t "$CONTAINER_NAME" update-alternatives --set php "/usr/bin/php${PHP_VERSION}"
  assert_success
  run docker exec -t "$CONTAINER_NAME" update-alternatives --set php-fpm "/usr/sbin/php-fpm${PHP_VERSION}"
  assert_success
}

@test "service runtime and log paths are writable without broadening base runtime directories for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker exec -t "$CONTAINER_NAME" bash -c 'test "$(stat -c %A /run | cut -c9)" != "w" && test -w /run/php && test -w /var/run/nginx && test -w /var/run/supervisor && test -w /var/run/apache2 && test -w /var/lock/apache2 && touch /var/log/apache2/ddev-runtime-write-test.log /var/log/nginx/ddev-runtime-write-test.log && printf test >> /var/log/php-fpm.log && printf test >> /var/log/supervisord.log'
  assert_success
}

@test "legacy PHP-FPM socket paths remain available for custom webserver configs for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker exec -t "$CONTAINER_NAME" bash -c 'test -L /run/php-fpm.sock && test "$(readlink /run/php-fpm.sock)" = "/run/php/php-fpm.sock" && test -S /run/php-fpm.sock && test -S /var/run/php-fpm.sock'
  assert_success
}

@test "enable and disable xdebug for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker exec -t $CONTAINER_NAME enable_xdebug
  assert_success
  if [[ ${PHP_VERSION} != 8.? ]] ; then
    run docker exec -t $CONTAINER_NAME php --re xdebug
    assert_success
    assert_output --partial "xdebug.remote_enable"
  else
    run docker exec -t $CONTAINER_NAME php --re xdebug
    assert_success
    assert_output --partial "xdebug.mode"
  fi
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php
  assert_success
  assert_output --partial "Xdebug is enabled"
  run docker exec -t $CONTAINER_NAME disable_xdebug
  assert_success
  run docker exec -t $CONTAINER_NAME php --re xdebug
  assert_failure
  assert_output --regexp "xdebug.*does not exist"
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xdebug.php
  assert_success
  assert_output --partial "Xdebug is disabled"
}

@test "enable and disable xhprof for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker exec -t $CONTAINER_NAME enable_xhprof
  assert_success
  run docker exec -t $CONTAINER_NAME php --re xhprof
  assert_success
  assert_output --partial "xhprof.output_dir"
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php
  assert_success
  assert_output --partial "XHProf is enabled"
  run docker exec -t $CONTAINER_NAME disable_xhprof
  assert_success
  run docker exec -t $CONTAINER_NAME php --re xhprof
  assert_failure
  assert_output --partial "does not exist"
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/test/xhprof.php
  assert_success
  assert_output --partial "XHProf is disabled"
}

@test "verify mailpit for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  if [ ${IS_HARDENED} == "true" ]; then skip "Skipping because mailpit is not installed on hardened prod image"; fi
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/test/test-email.php
  assert_success
  assert_output --partial "Test email sent"
  run curl -s --fail 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  assert_success
}

@test "verify PHP ini settings for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # Default settings for assert.active should be 1
  if [[ ${PHP_VERSION} != 8.? ]]; then
    run docker exec -t $CONTAINER_NAME php -i
    assert_success
    assert_output --regexp "assert.active.*=> 1 => 1"
  else
    run docker exec -t $CONTAINER_NAME php -i
    assert_success
    assert_output --regexp "assert.active.*=> On => On"
  fi
}

@test "verify phpstatus endpoint for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run curl -s 127.0.0.1:$HOST_HTTP_PORT/phpstatus
  assert_success
  assert_output --regexp "idle processes|php is working"
}

@test "verify error conditions for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # A request for a nonexistent path should still be a 404, but with
  # ddev-webserver's own informative message explaining that the 404 came
  # from the webserver layer (missing file/docroot), not the project's app.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/asdf
  assert_success
  assert_output --partial "404"
  assert_output --partial "generated by the ddev-webserver container"
  assert_output --partial "docroot"
  assert_output --partial "ddev logs"
  # The X-DDEV-404-Source header must be visible even on a HEAD/-I request,
  # so it's discoverable without inspecting the response body.
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/asdf
  assert_success
  assert_output --partial "X-DDEV-404-Source"
  # We're just checking the error code here - there's not much more we can do in
  # this case because the container is *NOT* intercepting 50x errors.
  for item in 400 401 500; do
    run curl -w "%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/${item}.php
    assert_success
    assert_output --partial "$item"
  done
}

@test "verify webserver 404s do not shadow app-generated 404 bodies for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # An existing PHP script that itself sends a 404 status with its own body
  # (as a framework's own "not found" route would) must be passed through
  # unchanged -- the ddev-webserver informative message must NOT appear.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/app-404.php
  assert_success
  assert_output --partial "404"
  assert_output --partial "App-level not found page"
  refute_output --partial "generated by the ddev-webserver container"
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/test/app-404.php
  assert_success
  refute_output --partial "X-DDEV-404-Source"
}

@test "verify that test/phptest.php is interpreted for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run curl --fail 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  assert_success
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
  8.0|8.1|8.2|8.3|8.4|8.5)
    extensions="$extensions memcached redis xdebug"
    ;;
  *)
    # Default fallback for future PHP versions - assume redis available
    extensions="$extensions redis"
    ;;
  esac

  # Load xhprof first, then xdebug, because loading xhprof disables xdebug
  run docker exec $CONTAINER_NAME enable_xhprof
  assert_success
  run docker exec $CONTAINER_NAME enable_xdebug
  assert_success
  run docker exec $CONTAINER_NAME bash -c "php -r 'foreach (get_loaded_extensions() as \$e) echo \$e, PHP_EOL;' 2>/dev/null"
  assert_success
  for item in $extensions; do
    assert_line "$item"
  done

  run docker exec $CONTAINER_NAME disable_xhprof
  assert_success
  run docker exec $CONTAINER_NAME disable_xdebug
  assert_success
  # disable_xdebug triggers an FPM reload; wait for it to settle before the next test
  sleep 2
}

@test "verify that both nginx logs and fpm logs are being tailed (${WEBSERVER_TYPE})" {
  run curl -sSL http://127.0.0.1:$HOST_HTTP_PORT/test/fatal.php
  assert_success
  # php-fpm message direct
  run bash -c "docker logs ${CONTAINER_NAME} 2>&1"
  assert_success
  assert_output --partial "PHP Fatal error:  Fatal error in"
}

@test "verify htaccess doesn't break ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  run docker cp tests/ddev-webserver/testdata/nginx/auth.conf ${CONTAINER_NAME}:/etc/nginx/common.d
  assert_success
  run docker cp tests/ddev-webserver/testdata/nginx/junkpass ${CONTAINER_NAME}:/tmp
  assert_success
  run docker cp tests/ddev-webserver/testdata/apache/auth.conf ${CONTAINER_NAME}:/etc/apache2/conf-enabled
  assert_success
  # Reload webserver
  if [ "${WEBSERVER_TYPE}" = "apache-fpm" ]; then
    run docker exec ${CONTAINER_NAME} apache2ctl -k graceful
    assert_success
  else
    run docker exec ${CONTAINER_NAME} nginx -s reload
    assert_success
  fi
  sleep 2
  # Make sure we can hit /phpstatus without auth
  run curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$HOST_HTTP_PORT/phpstatus
  assert_success
  assert_output "200"
  run curl --fail -s http://127.0.0.1:$HOST_HTTP_PORT/phpstatus
  assert_success
  assert_output --regexp "idle processes|php is working"
  # Make sure the auth requirement is actually working
  run curl --fail -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  assert_output "401"

  # Make sure it works with auth when hitting phptest.php
  AUTH=$(echo -ne "junk:junk" | base64)
  run curl --fail --header "Authorization: Basic $AUTH" 127.0.0.1:$HOST_HTTP_PORT/test/phptest.php
  assert_success
  run docker exec ${CONTAINER_NAME} rm /etc/nginx/common.d/auth.conf /etc/apache2/conf-enabled/auth.conf
  assert_success
  run docker exec ${CONTAINER_NAME} kill -HUP 1
  assert_success
  run docker exec ${CONTAINER_NAME} kill -USR2 1
  assert_success
}
