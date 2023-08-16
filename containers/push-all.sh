#!/bin/bash

set -eu -o pipefail
if [ "${VERSION:-}" = "" ]; then
  export VERSION=$(git describe --tags --always --dirty)
fi
for item in ddev-php-base ddev-nginx-proxy-router ddev-ssh-agent ddev-traefik-router ddev-webserver; do
  echo "=========== PUSHING $item:${VERSION} ============"
  pushd $item >/dev/null
  make push VERSION=${VERSION}
  popd >/dev/null
done

pushd ddev-dbserver
echo "=========== PUSHING ddev-dbserver:${VERSION} ============"
make PUSH=true VERSION=${VERSION}
