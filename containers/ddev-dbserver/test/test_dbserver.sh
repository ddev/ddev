#!/bin/bash

# Find the directory of this script
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"

set -o errexit
set -o pipefail
set -o nounset

if [ $# != 3 ]; then
  echo "Usage: $0 <db_type> <db_version> <tag>, for example $0 mariadb 10.3 v1.17.0"
  exit 1
fi
export DB_TYPE=$1
export DB_VERSION=$2
export TAG=$3
export IMAGE=drud/ddev-dbserver-${DB_TYPE}-${DB_VERSION}:${TAG}

export CURRENT_ARCH=$(../get_arch.sh)

# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"
bats test || (echo "bats tests failed for DB_TYPE ${DB_TYPE} DB_VERSION=${DB_VERSION} TAG=${TAG}" && exit 2)
printf "Test successful for DB_TYPE ${DB_TYPE} DB_VERSION=${DB_VERSION} TAG=${TAG}\n\n"
