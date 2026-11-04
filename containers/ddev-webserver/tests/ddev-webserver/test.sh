#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
export DOCKER_IMAGE=$1
export IS_HARDENED=false
DOCKER_REPO=${DOCKER_IMAGE%:*}
if [[ "${DOCKER_REPO}" == "*prod*" ]]; then
  IS_HARDENED=true
fi

# Find the directory of this script
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
export TEST_SCRIPT_DIR=${DIR}/../../../testscripts

export HOST_HTTP_PORT="8080"
export HOST_HTTPS_PORT="8443"
export CONTAINER_HTTP_PORT="80"
export CONTAINER_HTTPS_PORT="443"
export CONTAINER_NAME=webserver-test
export PHP_VERSION=8.4
export WEBSERVER_TYPE=nginx-fpm

export MOUNTUID=33
export MOUNTGID=33
# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"

mkcert -install
# docker cp makes the copied files root-owned, so the key must be world-readable
CA_COPY_DIR="$(mktemp -d)"
cp "$(mkcert -CAROOT)/rootCA.pem" "$(mkcert -CAROOT)/rootCA-key.pem" "$CA_COPY_DIR"
chmod 644 "$CA_COPY_DIR"/*

# Wait for container to be ready.
function containerwait {
  for _ in {20..0}; do
    status="$(docker inspect $CONTAINER_NAME | jq -r '.[0].State.Status')"
    health="$(docker inspect $CONTAINER_NAME | jq -r '.[0].State.Health.Status')"
    case $status in
    running)
      if [ "${health}" = "healthy" ]; then
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
      ;;
    esac
  done
  echo "# --- containerwait failed: information:"
  return 1
}

# Like DDEV, copy the host's CA into $CAROOT before the container starts, so
# start.sh signs the webserver's certificate with it, and add the test fixtures.
# Arguments go to docker create.
function runwebserver {
  docker create "$@" --name "$CONTAINER_NAME" "$DOCKER_IMAGE" >/dev/null
  docker cp "$CA_COPY_DIR/." "$CONTAINER_NAME:/usr/local/lib/ddev/mkcert"
  docker cp "$DIR/fixtures/test" "$CONTAINER_NAME:/var/www/html/test"
  docker start "$CONTAINER_NAME" >/dev/null
}

function cleanup {
  docker rm -f $CONTAINER_NAME >/dev/null 2>&1 || true
}

function finalcleanup {
  cleanup
  rm -rf "$CA_COPY_DIR"
}
trap finalcleanup EXIT
cleanup

# Run general tests with a default container
runwebserver -u "$MOUNTUID:$MOUNTGID" -p "$HOST_HTTP_PORT:$CONTAINER_HTTP_PORT" -p "$HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT" -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}"
if ! containerwait; then
  echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
  exit 100
fi
bats --show-output-of-passing-tests tests/ddev-webserver/general.bats

cleanup

for PHP_VERSION in 8.2 8.3 8.4 8.5 8.6; do
  for WEBSERVER_TYPE in nginx-fpm apache-fpm; do
    export PHP_VERSION WEBSERVER_TYPE DOCKER_IMAGE
    runwebserver -u "$MOUNTUID:$MOUNTGID" -p "$HOST_HTTP_PORT:$CONTAINER_HTTP_PORT" -p "$HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT" -e "DDEV_PHP_VERSION=${PHP_VERSION}" -e "DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE}"
    if ! containerwait; then
      echo "=============== Failed containerwait after docker run with  DDEV_WEBSERVER_TYPE=${WEBSERVER_TYPE} DDEV_PHP_VERSION=$PHP_VERSION ==================="
      exit 101
    fi
    bats tests/ddev-webserver/php_webserver.bats || ( echo "bats tests failed for WEBSERVER_TYPE=$WEBSERVER_TYPE PHP_VERSION=$PHP_VERSION" && exit 102 )
    printf "Test successful for PHP_VERSION=%s WEBSERVER_TYPE=%s\n\n" "$PHP_VERSION" "$WEBSERVER_TYPE"
    cleanup
  done
done

for project_type in backdrop craftcms drupal drupal7 drupal10 drupal11 laravel magento magento2 maho modx symfony typo3 wordpress default; do
  export PHP_VERSION="8.4"
  export project_type
  runwebserver -u "$MOUNTUID:$MOUNTGID" -p "$HOST_HTTP_PORT:$CONTAINER_HTTP_PORT" -p "$HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT" -e "DOCROOT=docroot" -e "DDEV_PHP_VERSION=$PHP_VERSION" -e "DDEV_PROJECT_TYPE=$project_type"
  if ! containerwait; then
    echo "=============== Failed containerwait after docker run with  DDEV_PROJECT_TYPE=${project_type} DDEV_PHP_VERSION=$PHP_VERSION ==================="
    exit 103
  fi
  bats tests/ddev-webserver/project_type.bats || ( echo "bats tests failed for project_type=$project_type" && exit 104 )
  printf "Test successful for project_type=%s\n\n" "$project_type"
  cleanup
done

runwebserver -u "$MOUNTUID:$MOUNTGID" -p "$HOST_HTTP_PORT:$CONTAINER_HTTP_PORT" -p "$HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT" -e "DDEV_PHP_VERSION=8.4" --mount "type=bind,src=$PWD/tests/ddev-webserver/testdata,target=/mnt/ddev_config"
containerwait

bats tests/ddev-webserver/custom_config.bats

cleanup

runwebserver -u "$MOUNTUID:$MOUNTGID" -p "$HOST_HTTP_PORT:$CONTAINER_HTTP_PORT" -p "$HOST_HTTPS_PORT:$CONTAINER_HTTPS_PORT" -e "DDEV_PHP_VERSION=8.4" --mount "type=bind,src=$PWD/tests/ddev-webserver/testdata,target=/mnt/ddev_config"
containerwait
bats tests/ddev-webserver/imagemagick.bats

cleanup

echo "Test successful"
