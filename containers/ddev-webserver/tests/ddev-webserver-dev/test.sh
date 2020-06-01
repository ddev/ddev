#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1
export HOST_HTTP_PORT="8080"
export HOST_HTTPS_PORT="8443"
export CONTAINER_HTTP_PORT="80"
export CONTAINER_HTTPS_PORT="443"
export CONTAINER_NAME=webserver-test
export PHP_VERSION=7.4
export WEBSERVER_TYPE=nginx-fpm

MOUNTUID=98
MOUNTGID=98
# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"

mkcert -install

# Wait for container to be ready.
function containerwait {
	for i in {60..0};
	do
		# status contains uptime and health in parenthesis, sed to return health
		status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
		if [[ "$status" == "healthy" ]]
		then
			return 0
		fi
		sleep 1
	done
	echo "# --- ddev-dbserver FAIL: information"
	docker ps -a
	docker logs $CONTAINER_NAME
	docker inspect $CONTAINER_NAME
	return 1
}

function cleanup {
	docker rm -f $CONTAINER_NAME >/dev/null 2>&1 || true
}
trap cleanup EXIT
cleanup

# We have to push the CA into the ddev-global-cache volume so it will be respected
docker run -t --rm -u "$MOUNTUID:$MOUNTGID" -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache $DOCKER_IMAGE bash -c "sudo mkdir -p /mnt/ddev-global-cache/mkcert && sudo chmod -R ugo+w /mnt/ddev-global-cache/* && sudo cp -R /mnt/mkcert /mnt/ddev-global-cache"

# Run general tests with a default container
docker run -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
if ! containerwait; then
    echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
    exit 100
fi
bats tests/ddev-webserver-dev/general.bats

cleanup

for PHP_VERSION in 5.6 7.0 7.1 7.2 7.3 7.4; do
    for WEBSERVER_TYPE in nginx-fpm apache-fpm apache-cgi; do
        export PHP_VERSION WEBSERVER_TYPE

        docker run -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
        if ! containerwait; then
            echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
            exit 101
        fi

        bats tests/ddev-webserver-dev/php_webserver.bats || ( echo "bats tests failed for WEBSERVER_TYPE=$WEBSERVER_TYPE PHP_VERSION=$PHP_VERSION" && exit 102 )
        printf "Test successful for PHP_VERSION=$PHP_VERSION WEBSERVER_TYPE=$WEBSERVER_TYPE\n\n"
        cleanup
    done
done

for project_type in drupal6 drupal7 drupal8 drupal9 typo3 backdrop wordpress default; do
	export PHP_VERSION="7.3"
    export project_type
	if [ "$project_type" == "drupal6" ]; then
	  PHP_VERSION="5.6"
	fi
	docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=$PHP_VERSION" -e "DDEV_PROJECT_TYPE=$project_type" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
    if ! containerwait; then
        echo "=============== Failed containerwait after docker run with  DDEV_PROJECT_TYPE=${project_type} DDEV_PHP_VERSION=$PHP_VERSION ==================="
        exit 103
    fi

    bats tests/ddev-webserver-dev/project_type.bats || ( echo "bats tests failed for project_type=$project_type" && exit 104 )
    printf "Test successful for project_type=$project_type\n\n"
    cleanup
done

docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=potato" -e "DDEV_PHP_VERSION=7.3" -v "/$PWD/tests/ddev-webserver-dev/testdata:/mnt/ddev_config:ro" -v ddev-global-cache:/mnt/ddev-global-cache -d --name $CONTAINER_NAME -d $DOCKER_IMAGE >/dev/null
containerwait

bats tests/ddev-webserver-dev/custom_config.bats

cleanup


echo "Test successful"
