#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

# Find the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

rm -f $DIR/../.git/hooks/pre-push

echo "Unlinked pre-push git hook"