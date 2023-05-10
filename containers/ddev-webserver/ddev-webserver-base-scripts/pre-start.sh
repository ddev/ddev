#!/bin/bash

# Create a pipe that other processes can use to get
# info into the docker logs
# The normal approach is for the other processes to write to
# /proc/1/fd/1, but that doesn't currently work on gitpod
# https://github.com/gitpod-io/gitpod/issues/17551

set -x
set -eu -o pipefail

logpipe=/var/tmp/logpipe
if [[ ! -p ${logpipe} ]]; then
    mkfifo ${logpipe}
fi

# Allow process 1 to be killed
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT SIGHUP

cat < ${logpipe}
