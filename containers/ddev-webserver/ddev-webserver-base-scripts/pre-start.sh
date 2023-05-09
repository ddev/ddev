#!/bin/bash

# Create a pipe that other processes can use to get
# info into the docker logs
# The normal approach is for the other processes to write to
# /proc/1/fd/1, but that doesn't currently work on gitpod
# https://github.com/gitpod-io/gitpod/issues/17551

set -x
set -eu -o pipefail

fifo=/var/tmp/logpipe
if [[ ! -p ${logpipe} ]]; then
    mkfifo ${logpipe}
fi

cat <${fifo}
