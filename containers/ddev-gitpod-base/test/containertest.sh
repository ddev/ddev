#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ "${OS:-$(uname)}" = "Windows_NT" ]; then exit; fi

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1
CONTAINER_NAME=ddev-gitpod-base-test

function cleanup {
  if [ "$(arch)" != "arm64" ] && [ "$(arch)" != "aarch64" ]; then
    echo "Removing $CONTAINER_NAME"
    docker rm -f $CONTAINER_NAME 2>/dev/null || true
  fi
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
  echo "# --- ddev-gitpod-base FAIL -----"
  return 1
}

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

if [ "$(arch)" != "arm64" ] && [ "$(arch)" != "aarch64" ]; then
  cleanup
  docker run -t --rm ${DOCKER_IMAGE} ddev version
fi