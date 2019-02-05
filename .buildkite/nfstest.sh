#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

function cleanup {
    docker volume rm nfstest >/dev/null || true
}
trap cleanup EXIT

OS=$(go env GOOS)

case $OS in
linux)
  share=/home
  ;;
darwin)
  share=/Users
  ;;
windows)
  share=/C/Users
  ;;
esac

# Find host.docker.internal name using host-docker-internal.sh script
hostDockerInternal=$($(dirname $0)/../scripts/host-docker-internal.sh)

docker volume create --driver local --opt type=nfs --opt o=addr=${hostDockerInternal},hard,nolock,rw --opt device=:${share} nfstest >/dev/null
docker run -t --rm -v nfstest:/tmp/nfs busybox ls //tmp/nfs >/dev/null
echo "nfsd seems to be set up ok"
