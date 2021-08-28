#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1
export IS_HARDENED=false
DOCKER_REPO=${DOCKER_IMAGE%:*}
if [[ "${DOCKER_REPO}" == "*prod*" ]]; then
  IS_HARDENED=true
fi

export HOST_HTTP_PORT="8080"
export HOST_HTTPS_PORT="8443"
export CONTAINER_HTTP_PORT="80"
export CONTAINER_HTTPS_PORT="443"
export CONTAINER_NAME=webserver-test
export PHP_VERSION=7.4
export WEBSERVER_TYPE=nginx-fpm

MOUNTUID=33
MOUNTGID=33
# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"

mkcert -install
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache busybox sh -c "mkdir -p /mnt/ddev-global-cache/mkcert && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache"


# Wait for container to be ready.
function containerwait {
	for i in {20..0};
	do
		status="$(docker inspect $CONTAINER_NAME | jq -r '.[0].State.Status')"
    health="$(docker inspect $CONTAINER_NAME | jq -r '.[0].State.Health.Status')"

		case $status in
		running)
      if [ ${health} = "healthy" ]; then
		    return 0
      else
        sleep 1
      fi
		  ;;
		exited)
		  echo "# --- container exited"
		  return 1
		  ;;
		*)
  		sleep 1
		esac
	done
	echo "# --- containerwait failed: information:"
	return 1
}

function cleanup {
	docker rm -f $CONTAINER_NAME >/dev/null 2>&1 || true
}
trap cleanup EXIT
cleanup

# We have to push the CA into the ddev-global-cache volume so it will be respected
docker run --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache ${DOCKER_IMAGE} bash -c "mkdir -p /mnt/ddev-global-cache/{mkcert,bashhistory,terminus} && cp -R /mnt/mkcert /mnt/ddev-global-cache/ && chown -Rf ${MOUNTUID}:${MOUNTGID} /mnt/ddev-global-cache/* && chmod -Rf ugo+w /mnt/ddev-global-cache/*"

# Run general tests with a default container
docker run -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
if ! containerwait; then
    echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
    exit 100
fi
bats tests/ddev-webserver/general.bats

cleanup

for PHP_VERSION in 5.6 7.0 7.1 7.2 7.3 7.4 8.0; do
    for WEBSERVER_TYPE in nginx-fpm apache-fpm; do
        export PHP_VERSION WEBSERVER_TYPE DOCKER_IMAGE
        docker run -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
        if ! containerwait; then
            echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
            exit 101
        fi

        bats tests/ddev-webserver/php_webserver.bats || ( echo "bats tests failed for WEBSERVER_TYPE=$WEBSERVER_TYPE PHP_VERSION=$PHP_VERSION" && exit 102 )
        printf "Test successful for PHP_VERSION=$PHP_VERSION WEBSERVER_TYPE=$WEBSERVER_TYPE\n\n"
        cleanup
    done
done

for project_type in backdrop drupal6 drupal7 drupal8 drupal9 laravel magento magento2 typo3 wordpress default; do
	export PHP_VERSION="7.4"
    export project_type
	if [ "$project_type" == "drupal6" ]; then
	  PHP_VERSION="5.6"
	fi
	docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=$PHP_VERSION" -e "DDEV_PROJECT_TYPE=$project_type" -d --name $CONTAINER_NAME -v ddev-global-cache:/mnt/ddev-global-cache -d $DOCKER_IMAGE >/dev/null
    if ! containerwait; then
        echo "=============== Failed containerwait after docker run with  DDEV_PROJECT_TYPE=${project_type} DDEV_PHP_VERSION=$PHP_VERSION ==================="
        exit 103
    fi

    bats tests/ddev-webserver/project_type.bats || ( echo "bats tests failed for project_type=$project_type" && exit 104 )
    printf "Test successful for project_type=$project_type\n\n"
    cleanup
done

docker run  -u "$MOUNTUID:$MOUNTGID" -p $HOST_HTTP_PORT:$CONTAINER_HTTP_PORT -p $HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT -e "DDEV_PHP_VERSION=7.4" --mount "type=bind,src=$PWD/tests/ddev-webserver/testdata,target=/mnt/ddev_config" -v ddev-global-cache:/mnt/ddev-global-cache -d --name $CONTAINER_NAME -d $DOCKER_IMAGE >/dev/null
containerwait

bats tests/ddev-webserver/custom_config.bats

cleanup


echo "Test successful"
