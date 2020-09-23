#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

if [ $# != "1" ]; then echo "Argument 'version' is required"; exit 1; fi
VERSION=$1

function cleanup {
    true
}
trap cleanup EXIT

export tag=${VERSION}
export DB_TYPE=mariadb
for v in 5.5 10.0 10.1 10.2 10.3 10.4 10.5; do
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$v:$tag"
    export DB_VERSION=$v
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $v" && exit 5 )
    printf "Test successful for $DB_TYPE $v\n\n"
done

export DB_TYPE=mysql
# Temporarily disable 5.6 tests due to https://github.com/drud/ddev/pull/2454#issuecomment-697137114
for v in 5.5 5.7 8.0; do
#for v in 5.5 5.6 5.7 8.0; do
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$v:$tag"
    export DB_VERSION=$v
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $v" && exit 5 )
    printf "Test successful for $DB_TYPE $v\n\n"
done

echo "Test successful"
