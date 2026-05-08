#!/usr/bin/env bash

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

# Kill process 1 + process group if this exist or fails
trap "trap - SIGTERM && kill -- -1" SIGINT SIGTERM EXIT SIGHUP SIGQUIT

# Run cat in background so bash can process signals during `wait`.
# If cat runs in the foreground, bash defers SIGTERM until cat exits,
# which never happens because nobody closes the write end of the pipe —
# causing Docker to wait the full stop_grace_period before sending SIGKILL.
cat < ${logpipe} &
wait
