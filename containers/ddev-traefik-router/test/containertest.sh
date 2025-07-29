#!/usr/bin/env bash

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
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ${SCRIPT_DIR}/testdata:/mnt/testdata -v ddev-global-cache:/mnt/ddev-global-cache ${DOCKER_IMAGE} bash -c "mkdir -p /mnt/ddev-global-cache/{mkcert,traefik} && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache && cp -rT /mnt/testdata/ /mnt/ddev-global-cache/traefik/"

# Run the router alone
docker run --rm --name $CONTAINER_NAME -p 8080:80 -p 8443:443 -v ddev-global-cache:/mnt/ddev-global-cache --name ${CONTAINER_NAME} -d $DOCKER_IMAGE --configFile=/mnt/ddev-global-cache/traefik/.static_config.yaml

if ! containercheck; then
    printf "=============== FAIL: $CONTAINER_NAME failed to become ready ====================\n"
    printf "=============== FAIL: $CONTAINER_NAME FAIL: information =================\n"
    docker logs $CONTAINER_NAME
    docker ps -a
    docker inspect $CONTAINER_NAME
    exit 101
fi

# Make sure internal access to monitor port is working
docker exec -t $CONTAINER_NAME bash -c 'traefik healthcheck --ping 2>&1' || { echo "Failed to run http healthcheck inside container" && exit 104; }

# Check that /api/http/routers endpoint is accessible and returns routers with providers
docker exec -t $CONTAINER_NAME bash -c "curl -sf http://127.0.0.1:\${TRAEFIK_MONITOR_PORT}/api/http/routers 2>&1 | jq -e 'length > 0' 2>&1" || { echo "Failed to get routers from /api/http/routers or no routers found" && exit 105; }

# Verify that all routers have a provider field
docker exec -t $CONTAINER_NAME bash -c "curl -sf http://127.0.0.1:\${TRAEFIK_MONITOR_PORT}/api/http/routers 2>&1 | jq -e 'all(has(\"provider\"))' 2>&1" || { echo "Some routers are missing provider field" && exit 106; }

# Check that /api/overview endpoint is accessible and has the required error fields
docker exec -t $CONTAINER_NAME bash -c "curl -sf http://127.0.0.1:\${TRAEFIK_MONITOR_PORT}/api/overview 2>&1 | jq -e 'has(\"http\") and .http.routers and .http.services and .http.middlewares' 2>&1" || { echo "Failed to get overview or missing http structure" && exit 107; }

# Verify that the error fields exist in the overview (they can be null/missing, but the parent objects should exist)
docker exec -t $CONTAINER_NAME bash -c "curl -sf http://127.0.0.1:\${TRAEFIK_MONITOR_PORT}/api/overview 2>&1 | jq -e '.http.routers and .http.services and .http.middlewares' 2>&1" || { echo "Missing required router/service/middleware sections in overview" && exit 108; }
