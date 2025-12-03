#!/usr/bin/env bash

# Find the directory of this script
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
export TEST_SCRIPT_DIR=${DIR}/../../testscripts

set -o errexit
set -o pipefail
set -o nounset

if [ $# -lt 3 ] || [ $# -gt 4 ]; then
  echo "Usage: $0 <db_type> <db_version> <tag> [test_file]"
  echo "Examples:"
  echo "  $0 mariadb 10.3 v1.17.0                    # Run all tests"
  echo "  $0 mariadb 10.3 v1.17.0 database_config    # Run only database_config.bats"
  exit 1
fi
export DB_TYPE=$1
export DB_VERSION=$2
export TAG=$3
export IMAGE=ddev/ddev-dbserver-${DB_TYPE}-${DB_VERSION}:${TAG}

# Determine which test(s) to run
if [ $# -eq 4 ]; then
  TEST_TARGET="test/${4}.bats"
  if [ ! -f "$TEST_TARGET" ]; then
    echo "Error: Test file $TEST_TARGET not found"
    exit 1
  fi
else
  TEST_TARGET="test"
fi

export CURRENT_ARCH=$(../get_arch.sh)

# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"
bats "$TEST_TARGET" || (echo "bats tests failed for DB_TYPE ${DB_TYPE} DB_VERSION=${DB_VERSION} TAG=${TAG}" && exit 2)
printf "Test successful for DB_TYPE ${DB_TYPE} DB_VERSION=${DB_VERSION} TAG=${TAG}\n\n"
