#!/usr/bin/env bats

@test "Verify required binaries are installed in normal image" {
    if [ "${IS_HARDENED}" == "true" ]; then skip "Skipping because IS_HARDENED==true"; fi
    COMMANDS="composer drush8 git magerun magerun2 mkcert node npm platform sudo terminus wp"
    for item in $COMMANDS; do
#      echo "# looking for $item" >&3
      docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null"
    done
}

@test "Verify some binaries binaries (sudo) are NOT installed in hardened image" {
    if [ "${IS_HARDENED}" != "true" ]; then skip "Skipping because IS_HARDENED==false"; fi
    COMMANDS="sudo terminus"
    for item in $COMMANDS; do
      rv=1
      docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null 2>/dev/null" || rv=$?
      if [ ${rv} -eq 0 ]; then echo "Found $item which should not be found" >/dev/stderr; return 1; fi
    done
}

@test "Verify /var/www/html/vendor/bin is in PATH on ddev/docker exec" {
    docker exec $CONTAINER_NAME bash -c 'echo $PATH | grep /var/www/html/vendor/bin'
}

@test "verify that xdebug is disabled by default when using start.sh to start" {
    docker exec $CONTAINER_NAME bash -c 'php --version | grep -v "with Xdebug"'
}

@test "verify that composer v2 is installed by default" {
    v=$(docker exec $CONTAINER_NAME bash -c 'composer --version | awk "{ print $3;}"')
    [[ "${v}" > "2." ]]
}

@test "verify that PHP assertion are enabled by default" {
    enabled=$(docker exec $CONTAINER_NAME sh -c "php -r 'echo ini_get("'"zend.assertions"'");'")
    [[ "${enabled}" -eq 1 ]]
}

@test "verify there aren't \"closed keepalive connection\" complaints" {
	(docker logs $CONTAINER_NAME 2>&1 | grep -v "closed keepalive connection")  || (echo "Found unwanted closed keepalive connection messages" && exit 103)
}

@test "verify access to upstream error messages ($project_type)" {
	ERRMSG="$(curl 127.0.0.1:$HOST_HTTP_PORT/test/upstream-error.php)"
	if [ "$ERRMSG" != "Upstream error message" ] ; then
	  exit 108
	fi
}


@test "verify apt keys are not expiring" {
  MAX_DAYS_BEFORE_EXPIRATION=90
  if [ "${DDEV_IGNORE_EXPIRING_KEYS:-}" = "true" ]; then
    skip "Skipping because DDEV_IGNORE_EXPIRING_KEYS is set"
  fi
  docker exec -e "max=$MAX_DAYS_BEFORE_EXPIRATION" ${CONTAINER_NAME} bash -c '
    dates=$(apt-key list 2>/dev/null | awk "/\[expires/ { gsub(/[\[\]]/, \"\"); print \$6;}")
    for item in ${dates}; do
      today=$(date -I)
      let diff=($(date +%s -d ${item})-$(date +%s -d ${today}))/86400
      if [ ${diff} -le ${max} ]; then
        exit 1
      fi
    done
  '
}
