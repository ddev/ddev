#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

OS=$(go env GOOS)

if [ "${OS}" = "windows" ]; then echo "host.docker.internal" && exit; fi

if [ "${OS}" = "darwin" ] ; then echo "host.docker.internal" && exit; fi

if [ "${OS}" = "linux" ]; then
    dockerIP=$(ip address show dev docker0 | awk  '$1 == "inet" { sub(/\/.*$/, "", $2); print $2 }')
    echo ${dockerIP}
    exit
fi

echo "Unable to determine host.docker.internal" && exit 101
