#!/bin/bash

# Find the directory of this script
export DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
export TEST_SCRIPT_DIR=${DIR}/../../testscripts

set -o errexit
set -o pipefail
set -o nounset

if [ $# != 1 ]; then
  echo "Usage: $0 <imagespec>"
  exit 1
fi
export IMAGE=$1

export CURRENT_ARCH=$(../get_arch.sh)

# /usr/local/bin is added for git-bash, where it may not be in the $PATH.
export PATH="/usr/local/bin:$PATH"
bats --show-output-of-passing-tests test || (echo "bats tests failed for IMAGE=${IMAGE}" && exit 2)
printf "Test successful for IMAGE=${IMAGE}\n\n"
