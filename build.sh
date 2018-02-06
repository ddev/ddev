#!/bin/bash

# This script is used to build drud/ddev using buildkite

function cleanEnvironment() {
	containers=`docker ps -a -q`
	if [ -n "$containers" ] ; then
	        docker stop $containers
	fi
	 
	# Delete all containers
	containers=`docker ps -a -q`
	if [ -n "$containers" ]; then
	        docker rm -f -v $containers
	fi
	 
	# Delete all images
	images=`docker images -q -a`
	if [ -n "$images" ]; then
	        docker rmi -f $images
	fi

	rm -rf ~/.ddev/Test*
}

# Manufacture a $GOPATH environment that can mount on docker.
export GOPATH=~/tmp/ddevbuild_$BUILDKITE_BUILD_ID
DRUDSRC=$GOPATH/src/github.com/drud
mkdir -p $DRUDSRC
ln -s $PWD $DRUDSRC/ddev
cd $DRUDSRC/ddev
echo "building $BUILDKITE_COMMIT at $(date) on $(hostname) for OS=$(go env GOOS) in $DRUDSRC/ddev"

export GOTEST_SHORT=1

cleanEnvironment

echo "Running tests..."
time make test

cleanEnvironment
