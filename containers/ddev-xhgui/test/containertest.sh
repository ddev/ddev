#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ $# != "1" ]; then echo "docker image spec must be \$1"; exit 1; fi
DOCKER_IMAGE=$1

# TODO: Add meaningful test here
true
