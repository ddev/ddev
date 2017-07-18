#!/bin/bash

# This script is used to build drud/ddev using surf 
# (https://github.com/surf-build/surf)

# Manufacture a $GOPATH environment that can mount on docker (when surf build)
if [ ! -z "$SURF_REF" ]; then
	BUILD=$(date "+%Y%m%d%H%M%S")
	export GOPATH=~/tmp/ddevbuild_$BUILD
	DRUDSRC=$GOPATH/src/github.com/drud
	mkdir -p $DRUDSRC
	ln -s $PWD $DRUDSRC/ddev
	cd $DRUDSRC/ddev
	echo "Surf building on $(hostname) for OS=$(go env GOOS) in $DRUDSRC/ddev"
fi

export GOTEST_SHORT=1

echo "Warning: deleting all docker containers"
docker rm -f $(docker ps -aq) 2>/dev/null || true


make testcmd
RV=$?
echo "build.sh completed with status=$RV"
exit $RV
