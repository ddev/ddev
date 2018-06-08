#!/bin/bash

# Runs "make test" in each container directory

set -o errexit
set -o pipefail
set -o nounset
set -x

# Make sure we don't have any existing containers on the testbot that might
# result in this container not being built from scratch.
VERSION=$(make version | sed 's/^VERSION://')
IMAGES=$(docker images | awk "/$VERSION/ { print \$3 }")
if [ ! -z "$IMAGES" ] ; then
  docker rmi --force $IMAGES
fi

for dir in containers/*
    do pushd $dir
    time make test
    popd
done
