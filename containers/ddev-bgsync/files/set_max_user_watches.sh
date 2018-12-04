#!/bin/bash

set -o pipefail
set -o errexit
set -o nounset
#set -x

# This requires first arg being the source directory
SYNC_SOURCE=${1:-/source}

file_count=$(ls -RA ${SYNC_SOURCE} | wc -l)
max_files=$((${file_count} + 10000))
echo "file_count in ${SYNC_SOURCE} is ${file_count}"

cur=$(sysctl fs.inotify.max_user_watches)

# If the current max_user_watches is less than the limit we want, set it higher
if [ "$cur" -lt "${max_files}" ]; then
    echo "setting max_files=${max_files}"
    sudo sysctl -w fs.inotify.max_user_watches=${max_files}
    echo "Set fs.inotify.max_user_watches=${max_files}"
fi

