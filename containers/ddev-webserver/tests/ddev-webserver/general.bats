#!/usr/bin/env bats

@test "Verify required binaries are installed in container" {
    COMMANDS="composer ddev-live drush git magerun magerun2 mkcert terminus wp"
    for item in $COMMANDS; do
       docker exec $CONTAINER_NAME bash -c "command -v $item >/dev/null"
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
