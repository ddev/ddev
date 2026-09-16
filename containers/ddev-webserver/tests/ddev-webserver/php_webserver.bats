#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-webserver/php_webserver.bats

setup() {
  load setup.sh
}

teardown() {
  # Backstop only: several tests below mutate container state and clean up
  # at their own end, which an aborted assert_* skips. Each check here is a
  # no-op for the tests that never dirtied that state, so this stays cheap.
  docker exec -u root ${CONTAINER_NAME} bash -c '
    reload_nginx=0
    if [ -f /mnt/ddev_config/nginx/error-pages.conf ]; then
      rm -f /mnt/ddev_config/nginx/error-pages.conf
      reload_nginx=1
    fi
    if [ -f /etc/nginx/common.d/auth.conf ]; then
      rm -f /etc/nginx/common.d/auth.conf
      reload_nginx=1
    fi
    [ "$reload_nginx" = "1" ] && pgrep -x nginx >/dev/null 2>&1 && nginx -s reload
    if [ -f /etc/apache2/conf-enabled/auth.conf ]; then
      rm -f /etc/apache2/conf-enabled/auth.conf
      pgrep -x apache2 >/dev/null 2>&1 && apache2ctl -k graceful
    fi
    php -m 2>/dev/null | grep -qix xdebug && disable_xdebug >/dev/null
    php -m 2>/dev/null | grep -qix xhprof && disable_xhprof >/dev/null
    true
  ' || true
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
  if [ "$status" -ne 0 ]; then
    # Xdebug isn't packaged for every PHP version (see php-packages.yaml); enable_xdebug
    # detects that at runtime and fails cleanly instead of pretending to enable it.
    assert_output --partial "Xdebug is unavailable for PHP ${PHP_VERSION}"
    run docker exec -t $CONTAINER_NAME php --re xdebug
    assert_failure
    assert_output --regexp "xdebug.*does not exist"
    return
  fi
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
  if [ "$status" -ne 0 ]; then
    # Xhprof isn't packaged for every PHP version (see php-packages.yaml); enable_xhprof
    # detects that at runtime and fails cleanly instead of pretending to enable it.
    assert_output --partial "Xhprof is unavailable for PHP ${PHP_VERSION}"
    run docker exec -t $CONTAINER_NAME php --re xhprof
    assert_failure
    assert_output --partial "does not exist"
    return
  fi
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
  # A nonexistent path must get ddev-webserver's explanatory 404, not a bare one.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/asdf
  assert_success
  assert_output --partial "404"
  assert_output --partial "ddev-webserver"
  assert_output --partial "docroot"
  assert_output --partial "ddev logs"
  # X-Ddev-404-Source must be visible without reading the body.
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/asdf
  assert_success
  assert_output --partial "X-Ddev-404-Source"
  # We're just checking the error code here - there's not much more we can do in
  # this case because the container is *NOT* intercepting 50x errors.
  for item in 400 401 500; do
    run curl -w "%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/${item}.php
    assert_success
    assert_output --partial "$item"
  done
}

@test "verify a directly-requested nonexistent .php file for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # A nonexistent .php file must be rejected before php-fpm (nginx
  # `try_files $uri =404`, apache `-f %{REQUEST_FILENAME}`), so it gets
  # ddev-webserver's 404 instead of php-fpm's "No input file specified".
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/this-does-not-exist.php
  assert_success
  assert_output --partial "404"
  assert_output --partial "ddev-webserver"
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/this-does-not-exist.php
  assert_success
  assert_output --partial "X-Ddev-404-Source"
}

@test "verify webserver 404s do not shadow app-generated 404 bodies for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # A script that sends its own 404 body must pass through unchanged.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/app-404.php
  assert_success
  assert_output --partial "404"
  assert_output --partial "App-level not found page"
  refute_output --partial "ddev-webserver"
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/test/app-404.php
  assert_success
  refute_output --partial "X-Ddev-404-Source"
}

@test "verify webserver 403 explanation for a no-index directory for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # test/ exists but holds only *.php helpers, so it has no index file. The
  # trailing slash makes it a directory request on both webservers.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/
  assert_success
  assert_output --partial "403"
  assert_output --partial "ddev-webserver"
  assert_output --partial "docroot"
  assert_output --partial "ddev logs"
  # Both webservers must serve the shared HTML page, not a stock 403 page.
  assert_output --partial "<title>403: Forbidden</title>"
  # X-Ddev-403-Source must be visible without reading the body.
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/test/
  assert_success
  assert_output --partial "403"
  assert_output --partial "X-Ddev-403-Source"
  assert_output --partial "Content-Type: text/html"
}

@test "verify webserver 403s do not shadow app-generated 403 bodies for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # A script that sends its own 403 body must pass through unchanged.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/app-403.php
  assert_success
  assert_output --partial "403"
  assert_output --partial "App-level forbidden page"
  refute_output --partial "ddev-webserver"
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/test/app-403.php
  assert_success
  refute_output --partial "X-Ddev-403-Source"
}

@test "verify webserver 403 explanation for a denied path for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # Both webservers deny dot-files, and deny before looking the file up, so the
  # path need not exist. These 403s reach the explanation too, which is why the
  # page names more than the missing-index case.
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/.htpasswd
  assert_success
  assert_output --partial "403"
  assert_output --partial "ddev-webserver"
  run curl -sI 127.0.0.1:$HOST_HTTP_PORT/test/.htpasswd
  assert_success
  assert_output --partial "X-Ddev-403-Source"
}

@test "verify directory listing is left to the project for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # The explanation must not take over mod_dir/mod_autoindex: a project that
  # turns listing on still gets a listing. nginx has no .htaccess, so its
  # autoindex is covered by the next test instead.
  if [ "${WEBSERVER_TYPE}" != "apache-fpm" ]; then
    skip "test/listing/.htaccess only applies to apache-fpm"
  fi
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/listing/
  assert_success
  assert_output --partial "200"
  assert_output --partial "listed-file.txt"
  refute_output --partial "ddev-webserver"
}

@test "verify a project nginx config outranks the explanations for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # Site configs include /mnt/ddev_config/nginx/*.conf before common.d, so a
  # project's own error_page wins; nginx serves the first matching one. Its
  # autoindex has to win as well, the way .htaccess does for apache above.
  if [ "${WEBSERVER_TYPE}" != "nginx-fpm" ]; then
    skip "project nginx config does not apply to ${WEBSERVER_TYPE}"
  fi
  run docker exec -u root ${CONTAINER_NAME} bash -c 'mkdir -p /mnt/ddev_config/nginx && printf "location /test/listing/ {\n    autoindex on;\n}\nerror_page 403 /test/app-403.php;\n" >/mnt/ddev_config/nginx/error-pages.conf && nginx -s reload'
  assert_success
  sleep 2
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/listing/
  assert_success
  assert_output --partial "200"
  assert_output --partial "listed-file.txt"
  run curl -s -w "\n%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/test/
  assert_success
  assert_output --partial "403"
  assert_output --partial "App-level forbidden page"
  refute_output --partial "ddev-webserver"
  run docker exec -u root ${CONTAINER_NAME} bash -c 'rm -f /mnt/ddev_config/nginx/error-pages.conf && nginx -s reload'
  assert_success
  sleep 2
}

@test "verify the explanation pages are not URLs of their own for ${WEBSERVER_TYPE} php${PHP_VERSION}" {
  # They answer as error documents only, so a project keeps these paths.
  for page in 403 404; do
    run curl -s -o /dev/null -w "%{http_code}" 127.0.0.1:$HOST_HTTP_PORT/ddev-webserver-${page}-error
    assert_success
    refute_output "200"
  done
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

  # /etc/php-packages.yaml is the single source of truth for which packages
  # are installed per PHP version/arch; drop any extension it doesn't list for
  # this version instead of hand-maintaining a per-version exclusion list here.
  arch=$(docker exec $CONTAINER_NAME dpkg --print-architecture)
  available=$(docker exec $CONTAINER_NAME yq ".php${PHP_VERSION//./}.${arch} | join(\" \")" /etc/php-packages.yaml)
  filtered=""
  for ext in $extensions; do
    pkg=$ext
    [ "$pkg" = "mysqli" ] && pkg="mysql"
    case " $available " in
    *" $pkg "*) filtered="$filtered $ext" ;;
    esac
  done
  extensions="$filtered"

  # Load xhprof first, then xdebug, because loading xhprof disables xdebug.
  # Either may be unavailable for this PHP version; ignore failure here since
  # the dedicated xhprof/xdebug tests already cover that behavior.
  docker exec $CONTAINER_NAME enable_xhprof || true
  docker exec $CONTAINER_NAME enable_xdebug || true
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
