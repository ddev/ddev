#!/bin/bash

# Find the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

set -o errexit
set -o pipefail
set -o nounset

if [ -z "$1" ]; then echo "Argument 'version' is required"; exit 1; fi
VERSION=$1

if [[ -n "${2-}" ]]; then
    ONLY_TESTS_FOR=$2
else
    ONLY_TESTS_FOR=
fi

function cleanup {
    true
}
trap cleanup EXIT

export tag=${VERSION}
export CURRENT_ARCH=$($DIR/../../get_arch.sh)

# Get database versions per database type
source $DIR/../database-versions

export MARIADB_VERSIONS="MARIADB_VERSIONS_$CURRENT_ARCH"
export MYSQL_VERSIONS="MYSQL_VERSIONS_$CURRENT_ARCH"

export DB_TYPE=mariadb
for v in ${!MARIADB_VERSIONS}; do
    if [[ "$ONLY_TESTS_FOR" == "mysql" ]]; then continue; fi;
    export baseversion=$(echo $v | awk -F ':' '{print $1}')
    export fullversion=$(echo $v | awk -F ':' '{print $2}')
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$baseversion:$tag"
    export DB_VERSION=$baseversion
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $baseversion" && exit 5 )
    printf "Test successful for $DB_TYPE $baseversion\n\n"
done

export DB_TYPE=mysql
for v in ${!MYSQL_VERSIONS}; do
    if [[ "$ONLY_TESTS_FOR" == "mariadb" ]]; then continue; fi;
    export baseversion=$(echo $v | awk -F ':' '{print $1}')
	export fullversion=$(echo $v | awk -F ':' '{print $2}')
    export IMAGE="drud/ddev-dbserver-$DB_TYPE-$baseversion:$tag"
    export DB_VERSION=$baseversion
    # /usr/local/bin is added for git-bash, where it may not be in the $PATH.
    export PATH="/usr/local/bin:$PATH"
    bats test || ( echo "bats tests failed for $DB_TYPE $baseversion" && exit 5 )
    printf "Test successful for $DB_TYPE $baseversion\n\n"
done

echo "Test successful"
