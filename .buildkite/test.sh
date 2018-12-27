#!/bin/bash

# This script is used to build drud/ddev using buildkite

echo "--- buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) for OS=$(go env GOOS) in $PWD with golang=$(go version) docker=$(docker version --format '{{.Server.Version}}') and docker-compose $(docker-compose version --short)"

export GOTEST_SHORT=1
export DRUD_NONINTERACTIVE=true

echo "--- cleaning up docker and Test directories"
echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
if [ "$(docker ps -aq | wc -l)" -gt 0 ] ; then
	docker rm -f $(docker ps -aq)
fi
docker system prune --volumes --force

# Update all images that could have changed
docker images | awk '/drud/ {print $1":"$2 }' | xargs -L1 docker pull

set -o errexit
set -o pipefail
set -o nounset
set -x

rm -rf ~/.ddev/Test*

# There are discrepancies in golang hash checking in 1.11+, so kill off modcache to solve.
# See https://github.com/golang/go/issues/27925
# This can probably be removed when current work is merged 2018-12-27
# go clean -modcache  (Doesn't work due to current bug in golang)
rm -rf ${GOPATH}/pkg/*

# Our testbot should now be sane, run the testbot checker to make sure.
./.buildkite/sanetestbot.sh

echo "Running tests..."
time make test
RV=$?
echo "build.sh completed with status=$RV"
exit $RV
