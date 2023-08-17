#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ "${OS:-$(uname)}" = "Windows_NT" ]; then exit; fi

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1
CONTAINER_NAME=ddev-traefik-router-test

function cleanup {
	echo "Removing $CONTAINER_NAME"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
}
trap cleanup EXIT

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
  echo "# --- ddev-traefik-router FAIL -----"
  return 1
}

cleanup

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Make sure rootCA is created and installed on the ddev-global-cache/mkcert
mkcert -install
set -x
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ${SCRIPT_DIR}/testdata:/mnt/testdata -v ddev-global-cache:/mnt/ddev-global-cache ${DOCKER_IMAGE} bash -c "mkdir -p /mnt/ddev-global-cache/{mkcert,traefik} && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache && cp /mnt/testdata/* /mnt/ddev-global-cache/traefik"

# Run the router alone
docker run --rm --name $CONTAINER_NAME -p 8080:80 -p 8443:443 -v ddev-global-cache:/mnt/ddev-global-cache --name ${CONTAINER_NAME} -d $DOCKER_IMAGE --configFile=/mnt/ddev-global-cache/traefik/static_config.yaml

if ! containercheck; then
    printf "=============== FAIL: $CONTAINER_NAME failed to become ready ====================\n"
    printf "=============== FAIL: $CONTAINER_NAME FAIL: information =================\n"
    docker logs $CONTAINER_NAME
    docker ps -a
    docker inspect $CONTAINER_NAME
    exit 101
fi

# Make sure we can access http and https ports successfully (and with valid cert)
#(curl -s -I http://127.0.0.1:8080 | grep 503) || (echo "Failed to get 503 from nginx-router by default" && exit 102)
## mkcert is not respected by git-bash curl, so don't try the test on windows.
#if [ "${OS:-$(uname)}" != "Windows_NT" ]; then
#    (curl -s -I https://127.0.0.1:8443 | grep 503) || (echo "Failed to get 503 from nginx-router via https by default" && exit 103)
#fi
# Make sure internal access to monitor port is working
docker exec -t $CONTAINER_NAME bash -c 'curl -I http://127.0.0.1:${TRAEFIK_MONITOR_PORT}/ping' || (echo "Failed to run http healthcheck inside container" && exit 104)
