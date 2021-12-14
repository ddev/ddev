#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

# Find the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

ln -sf $DIR/pre-push.gofmt $DIR/../.git/hooks/pre-push

echo "Linked pre-push.gofmt as pre-push git hook"