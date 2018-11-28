#!/bin/bash

set -x

set -o errexit
set -o pipefail
set -o nounset

HOST_PORT="1081"
CONTAINER_PORT="80"
CONTAINER_NAME=web-local-test
DOCKER_IMAGE=$(awk '{print $1}' .docker_image)

# Wait for container to be ready.
function containercheck {
    set +x
	for i in {10..0};
	do
		# status contains uptime and health in parenthesis, sed to return health
		status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
		if [[ "$status" == "healthy" ]]
		then
			set -x
			return 0
		fi
		sleep 1
	done
	echo "nginx container did not become ready"
	echo "--- FAIL: ddev-webserver container failure info"
    docker ps -a
    docker logs $CONTAINER_NAME
    docker inspect $CONTAINER_NAME
	return 1
}

function cleanup {
	echo "Removing $CONTAINER_NAME"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
}
trap cleanup EXIT

cleanup

UNAME=$(uname)

# Using a static composer dir saves the composer downloads for each php version.
composercache=${HOME}/tmp/composercache_${RANDOM}_$$
mkdir -p $composercache && chmod 777 $composercache

export MOUNTUID=$UID
export MOUNTGID=$(id -g)
if [ "$UNAME" = "MINGW64_NT-10.0" -o "$MOUNTUID" -ge 60000 ] ; then
	MOUNTUID=1000
	MOUNTGID=1000
fi

for v in 5.6 7.0 7.1 7.2 7.3; do
	echo "starting container for tests on php$v"

	docker run -u "$MOUNTUID:$MOUNTGID" -p $HOST_PORT:$CONTAINER_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=$v" -d --name $CONTAINER_NAME -v "/$composercache:/home/.composer/cache:rw" -d $DOCKER_IMAGE
	if ! containercheck; then
        exit 101
    fi

	curl --fail localhost:$HOST_PORT/test/phptest.php
	curl -s localhost:$HOST_PORT/test/test-email.php | grep "Test email sent"
	docker exec -t $CONTAINER_NAME php --version | grep "PHP $v"
	docker exec -t $CONTAINER_NAME drush --version
	docker exec -t $CONTAINER_NAME wp --version

	# Make sure composer create-project is working.
	docker exec -t $CONTAINER_NAME composer create-project -d //tmp drupal-composer/drupal-project:8.x-dev my-drupal8-site --stability dev --no-interaction

    # Default settings for assert.active should be 1
    docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 1 => 1"

	echo "testing error states for php$v"
	# These are just the standard nginx 403 and 404 pages
	curl localhost:$HOST_PORT/ | grep "403 Forbidden"
	curl localhost:$HOST_PORT/asdf | grep "404 Not Found"
	# We're just checking the error code here - there's not much more we can do in
	# this case because the container is *NOT* intercepting 50x errors.
	curl -w "%{http_code}" localhost:$HOST_PORT/test/500.php | grep 500
	# 400 and 401 errors are intercepted by the same page.
	curl localhost:$HOST_PORT/test/400.php | grep "ddev web container.*400"
	curl localhost:$HOST_PORT/test/401.php | grep "ddev web container.*401"

	echo "testing php and email for php$v"
	curl --fail localhost:$HOST_PORT/test/phptest.php
	curl -s localhost:$HOST_PORT/test/test-email.php | grep "Test email sent"

    # Make sure the phpstatus url is working for testing php-fpm.
    curl -s localhost:$HOST_PORT/phpstatus | grep "idle processes"

	docker rm -f $CONTAINER_NAME
done

# Run various project_types and check behavior.
for project_type in drupal6 drupal7 drupal8 typo3 backdrop wordpress default; do
	PHP_VERSION="7.1"

	if [ "$project_type" == "drupal6" ]; then
	  PHP_VERSION="5.6"
	fi
	docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_PORT:$CONTAINER_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=$PHP_VERSION" -e "DDEV_PROJECT_TYPE=$project_type" -d --name $CONTAINER_NAME -d $DOCKER_IMAGE
	if ! containercheck; then
        exit 102
    fi
	curl --fail localhost:$HOST_PORT/test/phptest.php
	# Make sure that the project-specific config has been linked in.
	docker exec -t $CONTAINER_NAME grep "# ddev $project_type config" //etc/nginx/nginx-site.conf
	# Make sure that the right PHP version was selected for the project_type
	# Only drupal6 is currently different here.
	docker exec -t $CONTAINER_NAME php --version | grep "PHP $PHP_VERSION"
	# xdebug should be disabled by default.
    docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug does not exist"

	# Make sure we don't have lots of "closed keepalive connection" complaints
	(docker logs $CONTAINER_NAME 2>&1 | grep -v "closed keepalive connection")  || (echo "Found unwanted closed keepalive connection messages" && exit 103)
	# Make sure both nginx logs and fpm logs are being tailed
    curl --fail localhost:$HOST_PORT/test/fatal.php
	(docker logs $CONTAINER_NAME 2>&1 | grep "WARNING:.* said into stderr:.*fatal.php on line " >/dev/null) || (echo "Failed to find WARNING: .pool www" && exit 104)
	(docker logs $CONTAINER_NAME 2>&1 | grep "FastCGI sent in stderr: .PHP message: PHP Fatal error:" >/dev/null) || (echo "failed to find FastCGI sent in stderr" && exit 105)

	# Make sure that backdrop drush commands were added on backdrop and only backdrop
	if [ "$project_type" == "backdrop" ] ; then
	 	# The .drush/commands/backdrop directory should only exist for backdrop apptype
		docker exec -t $CONTAINER_NAME bash -c 'if [ ! -d  ~/.drush/commands/backdrop ] ; then echo "Failed to find expected backdrop drush commands"; exit 106; fi'
	else
		docker exec -t $CONTAINER_NAME bash -c 'if [ -d  ~/.drush/commands/backdrop ] ; then echo "Found unexpected backdrop drush commands"; exit 107; fi'
	fi
	docker rm -f $CONTAINER_NAME
done

echo "--- testing use of custom nginx and php configs"
docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_PORT:$CONTAINER_PORT -e "DOCROOT=potato" -e "DDEV_PHP_VERSION=7.2" -v "/$PWD/test/testdata:/mnt/ddev_config:ro" -d --name $CONTAINER_NAME -d $DOCKER_IMAGE
if ! containercheck; then
    exit 108
fi
docker exec -t $CONTAINER_NAME grep "docroot is /var/www/html/potato in custom conf" //etc/nginx/sites-enabled/nginx-site.conf

# Enable xdebug (and then disable again) and make sure it does the right thing.
echo "--- Turn on and off xdebug and check the results
docker exec -t $CONTAINER_NAME enable_xdebug
docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug.remote_enable"
docker exec -t $CONTAINER_NAME disable_xdebug
docker exec -t $CONTAINER_NAME php --re xdebug | grep "xdebug does not exist"

# Verify that the custom php configuration in ddev_config/php is activated.
echo "--- Verify that /mnt/ddev_config is mounted and we have php overrides there.

# First see if /mnt/ddev_config "works"
docker exec -t $CONTAINER_NAME bash -c "ls -l //mnt/ddev_config/php/my-php.ini || (echo 'Failed to ls /mnt/ddev_config' && exit 201)"

# With overridden value we should have assert.active=0, not the default
echo "--- Check that assert.active override is working"
docker exec -t $CONTAINER_NAME php -i | grep "assert.active.*=> 0 => 0" >/dev/null
