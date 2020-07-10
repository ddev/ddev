#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

@test "verify that backdrop drush commands were added on backdrop and only backdrop ($project_type)" {
	if [ "$project_type" = "backdrop" ] ; then
	 	# The .drush/commands/backdrop directory should only exist for backdrop apptype
		docker exec -t $CONTAINER_NAME bash -c 'if [ ! -d  ~/.drush/commands/backdrop ] ; then echo "Failed to find expected backdrop drush commands"; exit 106; fi'
	else
		docker exec -t $CONTAINER_NAME bash -c 'if [ -d  ~/.drush/commands/backdrop ] ; then echo "Found unexpected backdrop drush commands"; exit 107; fi'
  fi
}

@test "verify that both nginx logs and fpm logs are being tailed ($project_type)" {
    curl --fail 127.0.0.1:$HOST_HTTP_PORT/test/fatal.php
	docker logs $CONTAINER_NAME 2>&1 | grep "WARNING:.* said into stderr:.*fatal.php on line " >/dev/null
	docker logs $CONTAINER_NAME 2>&1 | grep "FastCGI sent in stderr: .PHP message: PHP Fatal error:" >/dev/null
}
