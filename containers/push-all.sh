#!/bin/bash

set -eu -o pipefail
if [ "${VERSION:-}" = "" ]; then
  export VERSION=$(git describe --tags --always --dirty)
fi
for item in ddev-php-base ddev-router ddev-ssh-agent ddev-webserver; do
  pushd $item >/dev/null
  make push VERSION=${VERSION}
  popd >/dev/null
done

pushd ddev-dbserver
make PUSH=true VERSION=${VERSION}
