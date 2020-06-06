#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

@test "verify that docroot/test/phptest.php is interpreted ($project_type)" {
	curl --fail localhost:$HOST_HTTP_PORT/test/phptest.php
}

@test "verify that project-specific config has been linked in ($project_type)" {
	docker exec $CONTAINER_NAME bash -c "grep \"# ddev $project_type config\" /etc/nginx/nginx-site.conf"
}

@test "verify that the right default PHP version was selected for the project_type ($project_type)" {
	# Only drupal6 is currently different here.
	docker exec -t $CONTAINER_NAME php --version | grep "PHP $PHP_VERSION"
}

@test "verify that xdebug is disabled by default ($project_type)" {
	# xdebug should be disabled by default.
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug does not exist"
}

@test "verify there aren't \"closed keepalive connection\" complaints ($project_type)" {
	(docker logs $CONTAINER_NAME 2>&1 | grep -v "closed keepalive connection")  || (echo "Found unwanted closed keepalive connection messages" && exit 103)
}

@test "verify that both nginx logs and fpm logs are being tailed ($project_type)" {
    curl --fail localhost:$HOST_HTTP_PORT/test/fatal.php
	docker logs $CONTAINER_NAME 2>&1 | grep "WARNING:.* said into stderr:.*fatal.php on line " >/dev/null
	docker logs $CONTAINER_NAME 2>&1 | grep "FastCGI sent in stderr: .PHP message: PHP Fatal error:" >/dev/null
}

@test "verify that backdrop drush commands were added on backdrop and only backdrop ($project_type)" {
	if [ "$project_type" = "backdrop" ] ; then
	 	# The .drush/commands/backdrop directory should only exist for backdrop apptype
		docker exec -t $CONTAINER_NAME bash -c 'if [ ! -d  ~/.drush/commands/backdrop ] ; then echo "Failed to find expected backdrop drush commands"; exit 106; fi'
	else
		docker exec -t $CONTAINER_NAME bash -c 'if [ -d  ~/.drush/commands/backdrop ] ; then echo "Found unexpected backdrop drush commands"; exit 107; fi'
  fi
}

@test "verify access to upstream error messages ($project_type)" {
	ERRMSG="$(curl localhost:$HOST_HTTP_PORT/test/upstream-error.php)"
	if [ "$ERRMSG" != "Upstream error message" ] ; then
	  exit 108
	fi
}
