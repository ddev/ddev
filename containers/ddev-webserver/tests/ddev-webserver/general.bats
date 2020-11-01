#!/usr/bin/env bats

@test "Verify required binaries are installed in normal image" {
    if [ "${IS_HARDENED}" == "true" ]; then skip "Skipping because IS_HARDENED==true"; fi
    COMMANDS="composer ddev-live drush git magerun magerun2 mkcert sudo terminus wp"
    for item in $COMMANDS; do
       docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null"
    done
}

@test "Verify some binaries binaries (sudo) are NOT installed in hardened image" {
    if [ "${IS_HARDENED}" != "true" ]; then skip "Skipping because IS_HARDENED==false"; fi
    COMMANDS="ddev-live sudo terminus"
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

@test "verify there aren't \"closed keepalive connection\" complaints" {
	(docker logs $CONTAINER_NAME 2>&1 | grep -v "closed keepalive connection")  || (echo "Found unwanted closed keepalive connection messages" && exit 103)
}

@test "verify access to upstream error messages ($project_type)" {
	ERRMSG="$(curl 127.0.0.1:$HOST_HTTP_PORT/test/upstream-error.php)"
	if [ "$ERRMSG" != "Upstream error message" ] ; then
	  exit 108
	fi
}
