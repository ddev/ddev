#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

function cleanup {
    docker volume rm nfstest >/dev/null || true
}
trap cleanup EXIT

mkdir -p ~/.ddev

# Handle new macOS Catalina /System/Volumes/Data share path.
share=${HOME}/.ddev
if [ -d "/System/Volumes/Data${HOME}/.ddev" ] ; then
    share="/System/Volumes/Data${HOME}/.ddev"
fi

# Find host.docker.internal name using host-docker-internal.sh script
hostDockerInternal=$($(dirname $0)/../../scripts/host-docker-internal.sh)

docker volume create --driver local --opt type=nfs --opt o=addr=${hostDockerInternal},hard,nolock,rw --opt device=:${share} nfstest >/dev/null
docker run -t --rm -v nfstest:/tmp/nfs busybox ls //tmp/nfs >/dev/null
echo "nfsd seems to be set up ok"
