#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset
set -x

make -s staticrequired
