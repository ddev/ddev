#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

function cleanup {
    true
}
trap cleanup EXIT

export tag=20191007_many_mariadb
export DB_TYPE=mariadb
for v in 5.5 10.0 10.0 10.1 10.2 10.3 10.4; do
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$v:$tag"
    export DB_VERSION=$v
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $v" && exit 5 )
    printf "Test successful for $DB_TYPE $v\n\n"
done

export DB_TYPE=mysql
for v in 5.5 5.6 5.7 8.0; do
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$v:$tag"
    export DB_VERSION=$v
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $v" && exit 5 )
    printf "Test successful for $DB_TYPE $v\n\n"
done

echo "Test successful"
