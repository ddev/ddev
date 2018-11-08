#!/bin/bash

# This script is used to build drud/ddev using buildkite

# Manufacture a $GOPATH environment that can mount on docker (when buildkite build)
if [ ! -z "$BUILDKITE_JOB_ID" ]; then
	export GOPATH=~/tmp/buildkite/$BUILDKITE_JOB_ID
	DRUDSRC=$GOPATH/src/github.com/drud
	mkdir -p $DRUDSRC
	ln -s $PWD $DRUDSRC/ddev
	cd $DRUDSRC/ddev
	echo "--- buildkite building $BUILDKITE_JOB_ID at $(date) on $(hostname) for OS=$(go env GOOS) in $DRUDSRC/ddev"
fi

export GOTEST_SHORT=1

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

# Our testbot should now be sane, run the testbot checker to make sure.
./.buildkite/sanetestbot.sh

echo "Running tests..."
time make test
RV=$?
echo "build.sh completed with status=$RV"
exit $RV
