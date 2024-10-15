#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ "${OS:-$(uname)}" = "Windows_NT" ]; then exit; fi

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1

if [ "$(arch)" != "arm64" ] && [ "$(arch)" != "aarch64" ]; then
  docker run -t --rm ${DOCKER_IMAGE} ddev --version
fi