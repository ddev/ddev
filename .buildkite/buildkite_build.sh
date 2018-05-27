#!/bin/bash

# This script is used to build drud/ddev using surf 
# (https://github.com/surf-build/surf)

# Manufacture a $GOPATH environment that can mount on docker (when surf build)
if [ ! -z "$BUILDKITE_JOB_ID" ]; then
	export GOPATH=~/tmp/buildkite/$BUILDKITE_JOB_ID
	DRUDSRC=$GOPATH/src/github.com/drud
	mkdir -p $DRUDSRC
	ln -s $PWD $DRUDSRC/ddev
	cd $DRUDSRC/ddev
	echo "buildkite building $BUILDKITE_JOB_ID at $(date) on $(hostname) for OS=$(go env GOOS) in $DRUDSRC/ddev"
fi

export GOTEST_SHORT=1

echo "Warning: deleting all docker containers and deleting ~/.ddev/Test*"
if [ "$(docker ps -aq | wc -l)" -gt 0 ] ; then
	docker rm -f $(docker ps -aq)
fi
ddev list
echo "Docker ps -a:"
docker ps -a

# Update all images that may have changed
docker images |grep -v REPOSITORY | awk '{print $1":"$2 }' | xargs -L1 docker pull
rm -rf ~/.ddev/Test*

echo "Running tests..."
time make test
RV=$?
echo "build.sh completed with status=$RV"
exit $RV
