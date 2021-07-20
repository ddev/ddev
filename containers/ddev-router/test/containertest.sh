#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ "${OS:-$(uname)}" = "Windows_NT" ]; then exit; fi

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1
CONTAINER_NAME=ddev-router-test

# Wait for container to be ready.
function containercheck {
  for i in {15..0}; do
    # fail if we can't find the container
    if ! docker inspect ${CONTAINER_NAME} >/dev/null; then
      break
    fi

    status="$(docker inspect ${CONTAINER_NAME} | jq -r '.[0].State.Status')"
    if [ "${status}" != "running" ]; then
      break
    fi
    health="$(docker inspect --format '{{json .State.Health }}' ${CONTAINER_NAME} | jq -r .Status)"
    case ${health} in
    healthy)
      return 0
      ;;
    *)
      sleep 1
      ;;
    esac
  done
  echo "# --- ddev-router FAIL -----"
  return 1
}

function cleanup {
	echo "Removing $CONTAINER_NAME"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
}
trap cleanup EXIT

cleanup

# Make sure rootCA is created and installed on the ddev-global-cache/mkcert
mkcert -install
set -x
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache busybox sh -c "mkdir -p /mnt/ddev-global-cache/mkcert && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache"

# Run the router alone
docker run --rm --name $CONTAINER_NAME -p 8080:80 -p 8443:443 --mount "type=bind,src=/var/run/docker.sock,target=/tmp/docker.sock" -v ddev-global-cache:/mnt/ddev-global-cache --name ddev-router-test -d $DOCKER_IMAGE

CONTAINER_NAME=ddev-router-test

if ! containercheck; then
    printf "=============== FAIL: $CONTAINER_NAME failed to become ready ====================\n"
    printf "=============== FAIL: $CONTAINER_NAME FAIL: information =================\n"
    docker logs $CONTAINER_NAME
    docker ps -a
    docker inspect $CONTAINER_NAME
    exit 101
fi

# Make sure we can access http and https ports successfully (and with valid cert)
(curl -s -I http://127.0.0.1:8080 | grep 503) || (echo "Failed to get 503 from nginx-router by default" && exit 102)
# mkcert is not respected by git-bash curl, so don't try the test on windows.
if [ "${OS:-$(uname)}" != "Windows_NT" ]; then
    (curl -s -I https://127.0.0.1:8443 | grep 503) || (echo "Failed to get 503 from nginx-router via https by default" && exit 103)
fi
# Make sure internal access to https is working
docker exec -t $CONTAINER_NAME curl --fail https://127.0.0.1/healthcheck || (echo "Failed to run https healthcheck inside container" && exit 104)


MAX_DAYS_BEFORE_EXPIRATION=90
if [ "${DDEV_IGNORE_EXPIRING_KEYS:-}" = "true" ]; then
  echo "Skipping test of expiring keys because DDEV_IGNORE_EXPIRING_KEYS is set"
else
  docker exec -e "max=$MAX_DAYS_BEFORE_EXPIRATION" ${CONTAINER_NAME} bash -x -c '
    dates=$(apt-key list 2>/dev/null | awk "/\[expires/ { gsub(/[\[\]]/, \"\"); print \$6;}")
    for item in ${dates}; do
      today=$(date -I)
      let diff=($(date +%s -d ${item})-$(date +%s -d ${today}))/86400
      if [ ${diff} -le ${max} ]; then
        echo "An apt key is expiring in ${diff} days"
        apt-key list
        exit 1
      fi
    done
  ' || (echo "apt keys are expiring in container" && exit 105)
fi
