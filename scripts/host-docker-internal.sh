#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

OS=$(go env GOOS)
ISTOOLBOX=""
if [ "${DOCKER_TOOLBOX_INSTALL_PATH:-}" != "" ] && [ {$DOCKER_HOST} != "" ]; then
  ISTOOLBOX=true
fi

if [ "${OS}" = "windows" ] && [ "${ISTOOLBOX}" = "" ]; then echo "host.docker.internal" && exit; fi

if [ "${OS}" = "darwin" ] ; then echo "host.docker.internal" && exit; fi

if [ "${OS}" = "linux" ]; then
    dockerIP=$(ip address show dev docker0 | awk  '$1 == "inet" { sub(/\/.*$/, "", $2); print $2 }')
    echo ${dockerIP}
    exit
fi

if [ "${ISTOOLBOX}" != "" ] ; then
    hostDockerInternalIP=$(echo $DOCKER_HOST | awk -F '.' ' { printf ("%d.%d.%d.1", $1, $2, $3) }')
    echo ${hostDockerInternalIP} && exit
fi

echo "Unable to determine host.docker.internal" && exit 101
