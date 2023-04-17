#!/bin/bash
set -eu -o pipefail

# Try `DOCKER_TAG="YYYYMMDD" ./push.sh`

DOCKER_TAG=${DOCKER_TAG:-$(git describe --tags --always --dirty)}

DOCKER_REPO=${DOCKER_REPO:-ddev/ddev-gitpod-base:${DOCKER_TAG}}

echo "Pushing ${DOCKER_REPO}"
set -x
# Build only current architecture and load into docker
docker buildx build -t ${DOCKER_REPO} --push --platform=linux/amd64 .

echo "This was pushed with ${DOCKER_REPO}. For it to take effect, it must be changed here in .gitpod.yml and also in https://github.com/ddev/ddev-gitpod-launcher/blob/main/.gitpod.yml"
