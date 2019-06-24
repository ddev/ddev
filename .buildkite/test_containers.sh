#!/bin/bash

# Runs "make test" in each container directory

set -o errexit
set -o pipefail
set -o nounset

echo "--- buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) for OS=$(go env GOOS) in $PWD with golang=$(go version) docker=$(docker version --format '{{.Server.Version}}') and docker-compose $(docker-compose version --short)"


function cleanup {
    set +x
    echo "--- Cleanup docker"
    echo "Warning: deleting all docker containers and deleting images that match this build."
    if [ "$(docker ps -aq | wc -l)" -gt 0 ] ; then
        docker rm -f $(docker ps -aq) >/dev/null || true
    fi

	docker system prune --volumes --force

    # Make sure we don't have any existing containers on the testbot that might
    # result in this container not being built from scratch.
    VERSION=$(make version | sed 's/^VERSION://')
    IMAGES=$(docker images | awk "/$VERSION/ { print \$3 }")
    if [ ! -z "${IMAGES:-}" ] ; then
      docker rmi --force $IMAGES 2>&1 >/dev/null || true
    fi
}

# Now that we've got a container running, we need to make sure to clean up
# at the end of the test run, even if something fails.
trap cleanup EXIT

# Do initial cleanup of images that might not be needed; they'll be cleaned at exit as well.
cleanup

# There are discrepancies in golang hash checking in 1.11+, so kill off modcache to solve.
# See https://github.com/golang/go/issues/27925
# This can probably be removed when current work is merged 2018-12-27
# go clean -modcache  (Doesn't work due to current bug in golang)
chmod -R u+w ~/go/pkg && rm -rf ~/go/pkg/*

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"
bash $(dirname $0)/testbot_maintenance.sh

# Our testbot should now be sane, run the testbot checker to make sure.
./.buildkite/sanetestbot.sh

set -x

for dir in containers/*
    do pushd $dir
    echo "--- Build container $dir"
    time make container DOCKER_ARGS=--no-cache
    echo "--- Test container $dir"
    time make test
    popd
done
